package arenaserver

import (
	"encoding/json"
	"fmt"
	"runtime"
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
	bettererrors "github.com/xtuc/better-errors"
)

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

	tickdurations []int64

	// Game logic

	game commongame.GameInterface

	events chan interface{}

	gameIsRunning bool
}

func NewServer(host string, port int, orch arenaservertypes.ContainerOrchestrator, gameDescription types.GameDescriptionInterface, game commongame.GameInterface, arenaServerUUID string, mqClient mq.ClientInterface) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost()
		utils.Check(err, "Could not determine arena-server host/ip.")

		gamehost = host
	}

	tickspersec := gameDescription.GetTps()

	s := &Server{
		host:            gamehost,
		port:            port,
		arenaServerUUID: arenaServerUUID,
		tickspersec:     tickspersec,

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

		tickdurations: make([]int64, 0),

		///////////////////////////////////////////////////////////////////////
		// Game logic
		///////////////////////////////////////////////////////////////////////

		game:          game,
		gameIsRunning: false,

		events: make(chan interface{}),
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

	server.Log(EventLog{"Listen"})
	block := server.listen()

	server.Log(EventLog{"Starting agent containers"})
	err := server.startAgentContainers()

	if err != nil {
		return nil, bettererrors.NewFromString("Failed to start agent containers").With(err)
	}

	server.gameIsRunning = true

	server.AddTearDownCall(func() error {
		server.Log(EventLog{"Publish game state (" + server.arenaServerUUID + "stopped)"})

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
	server.gameIsRunning = false

	server.Log(EventDebug{"TearDown from stop"})
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

	s.Log(EventLog{"Send game launched: " + string(payloadJson)})
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
	server.Log(EventLog{"Agents are ready; starting in 1 second"})
	time.Sleep(time.Duration(time.Second * 1))

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
					server.Log(EventLog{"Received stop ticking signal"})
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

	begin := time.Now()

	turn := int(server.currentturn) // starts at 0
	atomic.AddUint32(&server.currentturn, 1)

	dolog := (turn % server.tickspersec) == 0

	if dolog {
		var totalDuration int64 = 0
		for _, duration := range server.tickdurations {
			totalDuration += duration
		}
		meanTick := float64(totalDuration) / float64(len(server.tickdurations))
		server.Log(EventStatusGameUpdate{fmt.Sprintf("Tick %d; %.3f ms mean; %d goroutines", turn, meanTick/1000000.0, runtime.NumGoroutine())})
	}

	///////////////////////////////////////////////////////////////////////////
	// Updating Game
	///////////////////////////////////////////////////////////////////////////

	timeStep := 1.0 / float64(server.GetTicksPerSecond())
	mutations := server.popMutationBatches()
	server.game.Step(turn, timeStep, mutations)

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

			if err != nil && server.gameIsRunning {
				berror := bettererrors.
					NewFromString("Failed to send perception").
					SetContext("agent", agentproxy.GetProxyUUID().String()).
					With(bettererrors.NewFromErr(err))

				server.Log(EventError{berror})
			}

		}(server, agentproxy, arenamap)
	}

	///////////////////////////////////////////////////////////////////////////
	// Pushing updated state to viz
	///////////////////////////////////////////////////////////////////////////

	notify.Post("app:stateupdated", nil)

	nbsamplesToKeep := server.GetTicksPerSecond() * 5
	if len(server.tickdurations) < nbsamplesToKeep {
		server.tickdurations = append(server.tickdurations, time.Now().UnixNano()-begin.UnixNano())
	} else {
		server.tickdurations[turn%nbsamplesToKeep] = time.Now().UnixNano() - begin.UnixNano()
	}
}

func (s *Server) AddTearDownCall(fn types.TearDownCallback) {
	s.tearDownCallbacksMutex.Lock()
	defer s.tearDownCallbacksMutex.Unlock()

	s.tearDownCallbacks = append(s.tearDownCallbacks, fn)
}

func (server *Server) TearDown() {
	server.events <- EventDebug{"teardown"}
	server.containerorchestrator.TearDownAll()

	server.tearDownCallbacksMutex.Lock()

	for i := len(server.tearDownCallbacks) - 1; i >= 0; i-- {
		server.events <- EventLog{"Executing TearDownCallback"}
		server.tearDownCallbacks[i]()
	}

	// Reset to avoid calling teardown callback multiple times
	server.tearDownCallbacks = make([]types.TearDownCallback, 0)

	server.tearDownCallbacksMutex.Unlock()

	server.events <- EventClose{}
}

func (server *Server) Events() chan interface{} {
	return server.events
}

func (server *Server) Log(l interface{}) {
	go func() {
		server.events <- l
	}()
}
