package arenaserver

import (
	"encoding/json"
	"errors"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	uuid "github.com/satori/go.uuid"

	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	commongame "github.com/bytearena/bytearena/game/common"
)

const debug = false

type Server struct {
	host            string
	port            int
	arenaServerUUID string
	tickspersec     int

	stopticking  chan bool
	nbhandshaked int
	currentturn  uint32

	tearDownCallbacks      []types.TearDownCallback
	tearDownCallbacksMutex *sync.Mutex

	containerorchestrator arenaservertypes.ContainerOrchestrator

	commserver *comm.CommServer
	mqClient   mq.ClientInterface

	agentimages            map[uuid.UUID]string
	agentproxies           map[uuid.UUID]agent.AgentProxyInterface
	agentproxiesmutex      *sync.Mutex
	agentproxieshandshakes map[uuid.UUID]struct{}

	pendingmutations []arenaservertypes.AgentMutationBatch
	mutationsmutex   *sync.Mutex

	gameDescription types.GameDescriptionInterface

	// Game logic

	game commongame.GameInterface
}

func NewServer(host string, port int, orch arenaservertypes.ContainerOrchestrator, gameDescription types.GameDescriptionInterface, game commongame.GameInterface, arenaServerUUID string, mqClient mq.ClientInterface) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost()
		utils.Check(err, "Could not determine arena-server host/ip.")

		gamehost = host
	}

	s := &Server{
		host:            gamehost,
		port:            port,
		arenaServerUUID: arenaServerUUID,
		tickspersec:     gameDescription.GetTps(),

		stopticking:  make(chan bool),
		nbhandshaked: 0,

		tearDownCallbacks:      make([]types.TearDownCallback, 0),
		tearDownCallbacksMutex: &sync.Mutex{},

		containerorchestrator: orch,
		commserver:            nil, // initialized in Listen()
		mqClient:              mqClient,

		gameDescription: gameDescription,

		// agents here: proxy to agent in container
		agentproxies:           make(map[uuid.UUID]agent.AgentProxyInterface),
		agentproxiesmutex:      &sync.Mutex{},
		agentproxieshandshakes: make(map[uuid.UUID]struct{}),
		agentimages:            make(map[uuid.UUID]string),

		pendingmutations: make([]arenaservertypes.AgentMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},

		///////////////////////////////////////////////////////////////////////
		// Game logic
		///////////////////////////////////////////////////////////////////////

		game: game,
	}

	return s
}

func (s Server) getNbExpectedagents() int {
	return len(s.GetGameDescription().GetContestants())
}

///////////////////////////////////////////////////////////////////////////////
// Public API
///////////////////////////////////////////////////////////////////////////////

func (server *Server) Start() (chan interface{}, error) {

	utils.Debug("arena", "Listen")
	block := server.listen()

	utils.Debug("arena", "Starting agent containers")
	err := server.startAgentContainers()

	if err != nil {
		return nil, errors.New("Failed to start agent containers: " + err.Error())
	}

	server.AddTearDownCall(func() error {
		utils.Debug("arena", "Publish game state ("+server.arenaServerUUID+"stopped)")

		game := server.GetGameDescription()

		err := server.mqClient.Publish("game", "stopped", types.NewMQMessage(
			"arena-server",
			"Arena Server "+server.arenaServerUUID+", game "+game.GetId()+" stopped",
		).SetPayload(types.MQPayload{
			"id":              game.GetId(),
			"arenaserveruuid": server.arenaServerUUID,
		}))

		return err
	})

	return block, nil
}

func (server *Server) Stop() {
	utils.Debug("arena-server", "TearDown from stop")
	server.TearDown()
}

func (s *Server) SubscribeStateObservation() chan interface{} {
	ch := make(chan interface{})
	notify.Start("app:stateupdated", ch)
	return ch
}

func (s *Server) SendLaunched() {
	payload := types.MQPayload{
		"id":              s.GetGameDescription().GetId(),
		"arenaserveruuid": s.arenaServerUUID,
	}

	s.mqClient.Publish("game", "launched", types.NewMQMessage(
		"arena-server",
		"Arena Server "+s.arenaServerUUID+" launched",
	).SetPayload(payload))

	payloadJson, _ := json.Marshal(payload)

	utils.Debug("arena-server", "Send game launched: "+string(payloadJson))
}

func (s Server) GetGameDescription() types.GameDescriptionInterface {
	return s.gameDescription
}

func (s Server) GetGame() commongame.GameInterface {
	return s.game
}

func (s Server) GetTicksPerSecond() int {
	return s.tickspersec
}

func (server *Server) onAgentsReady() {
	utils.Debug("arena", "Agents are ready; starting in 1 second")
	time.Sleep(time.Duration(time.Second * 1))

	go func() {
		stopChannel := make(chan bool)
		server.monitoring(stopChannel)

		server.AddTearDownCall(func() error {
			stopChannel <- true
			return nil
		})
	}()

	server.startTicking()
}

func (server *Server) startTicking() {

	tickduration := time.Duration((1000000 / time.Duration(server.tickspersec)) * time.Microsecond)
	ticker := time.Tick(tickduration)

	server.AddTearDownCall(func() error {
		server.stopticking <- true
		close(server.stopticking)

		return nil
	})

	go func() {

		for {
			select {
			case <-server.stopticking:
				{
					utils.Debug("core-loop", "Received stop ticking signal")
					notify.Post("app:stopticking", nil)
					break
				}
			case <-ticker:
				{
					server.doTick()
				}
			}
		}
	}()
}

func (server *Server) popMutationBatches() []arenaservertypes.AgentMutationBatch {
	server.mutationsmutex.Lock()
	mutations := server.pendingmutations
	server.pendingmutations = make([]arenaservertypes.AgentMutationBatch, 0)
	server.mutationsmutex.Unlock()

	return mutations
}

func (server *Server) doTick() {

	turn := int(server.currentturn)
	atomic.AddUint32(&server.currentturn, 1)

	dolog := (turn % server.tickspersec) == 0

	if dolog {
		utils.Debug("core-loop", "######## Tick ######## "+strconv.Itoa(turn))
		utils.Debug("core-loop", "Goroutines in flight : "+strconv.Itoa(runtime.NumGoroutine()))
	}

	///////////////////////////////////////////////////////////////////////////
	// Updating Game
	///////////////////////////////////////////////////////////////////////////

	timeStep := 1.0 / float64(server.GetTicksPerSecond())
	mutations := server.popMutationBatches()
	server.game.Step(timeStep, mutations)

	///////////////////////////////////////////////////////////////////////////
	// Refreshing perception for every agent
	///////////////////////////////////////////////////////////////////////////

	arenamap := server.GetGameDescription().GetMapContainer()
	for _, agentproxy := range server.agentproxies {
		go func(server *Server, agentproxy agent.AgentProxyInterface, arenamap *mapcontainer.MapContainer) {

			err := agentproxy.SetPerception(
				server.GetGame().GetAgentPerception(agentproxy.GetEntityId()),
				server,
			)
			if err != nil {
				utils.Debug("arenaserver", "ERROR: could not set perception on agent "+agentproxy.GetProxyUUID().String())
			}

		}(server, agentproxy, arenamap)
	}

	///////////////////////////////////////////////////////////////////////////
	// Pushing updated state to viz
	///////////////////////////////////////////////////////////////////////////

	notify.Post("app:stateupdated", nil)
}

func (s *Server) AddTearDownCall(fn types.TearDownCallback) {
	s.tearDownCallbacksMutex.Lock()
	defer s.tearDownCallbacksMutex.Unlock()

	s.tearDownCallbacks = append(s.tearDownCallbacks, fn)
}

func (server *Server) TearDown() {
	utils.Debug("arena", "teardown")
	server.containerorchestrator.TearDownAll()

	server.tearDownCallbacksMutex.Lock()

	for i := len(server.tearDownCallbacks) - 1; i >= 0; i-- {
		utils.Debug("teardown", "Executing TearDownCallback")
		server.tearDownCallbacks[i]()
	}

	// Reset to avoid calling teardown callback multiple times
	server.tearDownCallbacks = make([]types.TearDownCallback, 0)

	server.tearDownCallbacksMutex.Unlock()
}
