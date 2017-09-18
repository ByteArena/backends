package arenaserver

import (
	"encoding/json"
	"errors"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/bytearena/bytearena/arenaserver/collision"
	"github.com/bytearena/bytearena/arenaserver/projectile"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	"github.com/bytearena/bytearena/arenaserver/container"
	"github.com/bytearena/bytearena/arenaserver/perception"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

const debug = false

type Server struct {
	host                  string
	arenaServerUUID       string
	port                  int
	stopticking           chan bool
	tickspersec           int
	containerorchestrator container.ContainerOrchestrator
	agents                map[uuid.UUID]agent.AgentInterface
	agentsmutex           *sync.Mutex
	state                 *state.ServerState
	commserver            *comm.CommServer
	nbhandshaked          int
	currentturn           utils.Tickturn
	currentturnmutex      *sync.Mutex
	stateobservers        []chan state.ServerState
	debugNbMutations      int
	debugNbUpdates        int
	mqClient              mq.ClientInterface

	agentimages map[uuid.UUID]string

	agenthandshakes map[uuid.UUID]struct{}

	game GameInterface

	tearDownCallbacks      []types.TearDownCallback
	tearDownCallbacksMutex *sync.Mutex
}

func NewServer(host string, port int, orch container.ContainerOrchestrator, game GameInterface, arenaServerUUID string, mqClient mq.ClientInterface) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost(&orch)
		utils.Check(err, "Could not determine arena-server host/ip.")

		gamehost = host
	}

	s := &Server{
		host:                  gamehost,
		arenaServerUUID:       arenaServerUUID,
		port:                  port,
		stopticking:           make(chan bool),
		tickspersec:           game.GetTps(),
		containerorchestrator: orch,
		agents:                make(map[uuid.UUID]agent.AgentInterface),
		agentsmutex:           &sync.Mutex{},
		state:                 state.NewServerState(game.GetMapContainer()),
		commserver:            nil, // initialized in Listen()
		nbhandshaked:          0,
		currentturnmutex:      &sync.Mutex{},
		mqClient:              mqClient,

		agentimages: make(map[uuid.UUID]string),

		agenthandshakes: make(map[uuid.UUID]struct{}),

		game: game,

		tearDownCallbacks:      make([]types.TearDownCallback, 0),
		tearDownCallbacksMutex: &sync.Mutex{},
	}

	return s
}

///////////////////////////////////////////////////////////////////////////////
// Public API
///////////////////////////////////////////////////////////////////////////////

func (s *Server) AddTearDownCall(fn types.TearDownCallback) {
	s.tearDownCallbacksMutex.Lock()
	defer s.tearDownCallbacksMutex.Unlock()

	s.tearDownCallbacks = append(s.tearDownCallbacks, fn)
}

func (server *Server) GetTicksPerSecond() int {
	return server.tickspersec
}

func (server *Server) RegisterAgent(agentimage, agentname string) {
	arenamap := server.game.GetMapContainer()
	agentSpawnPointIndex := len(server.agents)

	if agentSpawnPointIndex >= len(arenamap.Data.Starts) {
		utils.Debug("arena", "Agent "+agentimage+" cannot spawn, no starting point left")
		return
	}

	agentSpawningPos := arenamap.Data.Starts[agentSpawnPointIndex]

	agent := agent.MakeNetAgentImp()
	agentstate := state.MakeAgentState(agent.GetId(), agentname, agentSpawningPos)

	server.setAgent(agent)
	server.state.SetAgentState(agent.GetId(), agentstate)

	server.agentimages[agent.GetId()] = agentimage

	utils.Debug("arena", "Registrer agent "+agentimage)
}

func (server *Server) GetState() *state.ServerState {
	return server.state
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

/* <implementing protocol.AgentCommunicator> */

func (server *Server) NetSend(message []byte, conn net.Conn) error {
	return server.commserver.Send(message, conn)
}

func (server *Server) PushMutationBatch(batch protocol.StateMutationBatch) {
	server.state.PushMutationBatch(batch)
	server.processMutations()
}

/* </implementing protocol.AgentCommunicator> */

func (server *Server) DispatchAgentMessage(msg protocol.MessageWrapperInterface) error {

	ag, err := server.doFindAgent(msg.GetAgentId().String())
	if err != nil {
		return errors.New("DispatchAgentMessage: agentid does not match any known agent in received agent message !;" + msg.GetAgentId().String())
	}

	// proto := msg.GetEmitterConn().LocalAddr().Network()
	// ip := strings.Split(msg.GetEmitterConn().RemoteAddr().String(), ":")[0]
	// if proto != "tcp" || ip != "TODO(jerome):take from agent container struct"
	// Problem here: cannot check ip against the one we get from Docker by inspecting the container
	// as the two addresses do not match

	switch msg.GetType() {
	case "Handshake":
		{
			if _, found := server.agenthandshakes[msg.GetAgentId()]; found {
				return errors.New("ERROR: Received duplicate handshake from agent " + ag.String())
			}

			server.agenthandshakes[msg.GetAgentId()] = struct{}{}

			var handshake protocol.MessageHandshakeImp
			err = json.Unmarshal(msg.GetPayload(), &handshake)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent handshake payload for agent " + msg.GetAgentId().String() + "; " + string(msg.GetPayload()))
			}

			ag, ok := ag.(agent.NetAgentInterface)
			if !ok {
				return errors.New("DispatchAgentMessage: Failed to cast agent to NetAgent during handshake for " + ag.String())
			}

			ag = ag.SetConn(msg.GetEmitterConn())
			server.setAgent(ag)

			utils.Debug("arena", "Received handshake from agent "+ag.String()+"; agent said \""+handshake.GetGreetings()+"\"")

			server.nbhandshaked++

			if server.nbhandshaked == server.getNbExpectedagents() {
				server.onAgentsReady()
			}

			// TODO(sven|jerome): handle some timeout here if all agents fail to handshake

			break
		}
	case "Mutation":
		{
			//break
			var mutations protocol.MessageMutationsImp
			err = json.Unmarshal(msg.GetPayload(), &mutations)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent mutation payload for agent " + ag.String() + "; " + string(msg.GetPayload()))
			}

			mutationbatch := protocol.StateMutationBatch{
				AgentId:   ag.GetId(),
				Mutations: mutations.GetMutations(),
			}

			server.PushMutationBatch(mutationbatch)

			break
		}
	default:
		{
			return errors.New("DispatchAgentMessage: Unknown message type" + msg.GetType())
		}
	}

	return nil
}

func (server *Server) Start() (chan interface{}, error) {

	utils.Debug("arena", "Listen")
	block := server.listen()

	utils.Debug("arena", "Spawn agents")
	err := server.spawnAgents()

	if err != nil {
		return nil, errors.New("Failed to spawn agents: " + err.Error())
	}

	server.AddTearDownCall(func() error {
		utils.Debug("arena", "Publish game state ("+server.arenaServerUUID+"stopped)")

		game := server.GetGame()

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

func (server *Server) SubscribeStateObservation() chan state.ServerState {
	ch := make(chan state.ServerState)
	server.stateobservers = append(server.stateobservers, ch)
	return ch
}

func (server *Server) SendLaunched() {
	payload := types.MQPayload{
		"id":              server.GetGame().GetId(),
		"arenaserveruuid": server.arenaServerUUID,
	}

	server.mqClient.Publish("game", "launched", types.NewMQMessage(
		"arena-server",
		"Arena Server "+server.arenaServerUUID+" launched",
	).SetPayload(payload))

	payloadJson, _ := json.Marshal(payload)

	utils.Debug("arena-server", "Send game launched: "+string(payloadJson))
}

func (server *Server) GetGame() GameInterface {
	return server.game
}

///////////////////////////////////////////////////////////////////////////////
// Private scope
///////////////////////////////////////////////////////////////////////////////

func (server *Server) spawnAgents() error {

	for _, agent := range server.agents {
		dockerimage := server.agentimages[agent.GetId()]

		arenaHostnameForAgents, err := server.containerorchestrator.GetHost(&server.containerorchestrator)
		if err != nil {
			return errors.New("Failed to fetch arena hostname for agents; " + err.Error())
		}

		container, err := server.containerorchestrator.CreateAgentContainer(agent.GetId(), arenaHostnameForAgents, server.port, dockerimage)

		if err != nil {
			return errors.New("Failed to create docker container for " + agent.String() + ": " + err.Error())
		}

		err = server.containerorchestrator.StartAgentContainer(container, server.AddTearDownCall)

		if err != nil {
			return errors.New("Failed to start docker container for " + agent.String() + ": " + err.Error())
		}

		server.AddTearDownCall(func() error {
			server.containerorchestrator.TearDown(container)

			return nil
		})
	}

	return nil
}

func (server *Server) setAgent(agent agent.AgentInterface) {
	server.agentsmutex.Lock()
	defer server.agentsmutex.Unlock()

	server.agents[agent.GetId()] = agent
}

func (s *Server) setTurn(turn utils.Tickturn) {
	s.currentturnmutex.Lock()
	defer s.currentturnmutex.Unlock()

	s.currentturn = turn
}

func (s *Server) getTurn() utils.Tickturn {
	s.currentturnmutex.Lock()
	defer s.currentturnmutex.Unlock()

	res := s.currentturn

	return res
}

func (server *Server) listen() chan interface{} {
	serveraddress := "0.0.0.0:" + strconv.Itoa(server.port)
	server.commserver = comm.NewCommServer(serveraddress)

	utils.Debug("arena", "Server listening on port "+strconv.Itoa(server.port))

	err := server.commserver.Listen(server)
	utils.Check(err, "Failed to listen on "+serveraddress)

	block := make(chan interface{})
	notify.Start("app:stopticking", block)

	return block
}

func (server *Server) getNbExpectedagents() int {
	return len(server.game.GetContestants())
}

func (server *Server) doFindAgent(agentid string) (agent.AgentInterface, error) {
	var emptyagent agent.AgentInterface

	foundkey, err := uuid.FromString(agentid)
	if err != nil {
		return emptyagent, err
	}

	server.agentsmutex.Lock()
	if foundagent, ok := server.agents[foundkey]; ok {
		server.agentsmutex.Unlock()
		return foundagent, nil
	}
	server.agentsmutex.Unlock()

	return emptyagent, errors.New("Agent" + agentid + " not found")
}

func (server *Server) doTick() {

	turn := server.getTurn()
	server.setTurn(turn.Next())

	dolog := (turn.GetSeq() % server.tickspersec) == 0

	if dolog {
		utils.Debug("core-loop", "######## Tick ######## "+strconv.Itoa(turn.GetSeq()))
	}

	///////////////////////////////////////////////////////////////////////////
	// Updating world state
	///////////////////////////////////////////////////////////////////////////
	server.update()

	///////////////////////////////////////////////////////////////////////////
	// Refreshing perception for every agent
	///////////////////////////////////////////////////////////////////////////
	server.GetState().DebugPoints = make([]vector.Vector2, 0)

	arenamap := server.game.GetMapContainer()

	for _, ag := range server.agents {
		go func(server *Server, ag agent.AgentInterface, serverstate *state.ServerState, arenamap *mapcontainer.MapContainer) {

			err := ag.SetPerception(
				perception.ComputeAgentPerception(arenamap, serverstate, ag),
				server,
			)
			if err != nil {
				utils.Debug("arenaserver", "ERROR: could not set perception on agent "+ag.GetId().String())
			}

		}(server, ag, server.GetState(), arenamap)
	}

	///////////////////////////////////////////////////////////////////////////
	// Pushing updated state to viz
	///////////////////////////////////////////////////////////////////////////
	serverCloned := *server.state

	for _, subscriber := range server.stateobservers {
		go func(s chan state.ServerState) {
			s <- serverCloned
		}(subscriber)
	}

	///////////////////////////////////////////////////////////////////////////

	if dolog {
		// Debug : Nombre de goroutines
		utils.Debug("core-loop", "Goroutines in flight : "+strconv.Itoa(runtime.NumGoroutine()))
	}
}

func (server *Server) monitoring(stopChannel chan bool) {
	monitorfreq := time.Second
	debugNbMutations := 0
	debugNbUpdates := 0
	for {
		select {
		case <-stopChannel:
			{
				break
			}
		case <-time.After(monitorfreq):
			{
				utils.Debug("monitoring",
					"-- MONITORING -- "+
						strconv.Itoa(server.debugNbMutations-debugNbMutations)+" mutations per "+monitorfreq.String()+";"+
						strconv.Itoa(server.debugNbUpdates-debugNbUpdates)+" updates per "+monitorfreq.String(),
				)

				debugNbMutations = server.debugNbMutations
				debugNbUpdates = server.debugNbUpdates

			}
		}
	}
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

func (server *Server) processMutations() {
	server.debugNbMutations++
	server.state.ProcessMutations()
}

func (server *Server) update() {

	server.debugNbUpdates++

	///////////////////////////////////////////////////////////////////////////
	// Updates physiques, liées au temps qui passe
	// Avant de récuperer les mutations de chaque tour, et même avant de constituer la perception de chaque agent
	///////////////////////////////////////////////////////////////////////////

	//
	// Updating projectiles
	//

	projectilesMovements := updateProjectiles(server)

	//
	// Updating agents
	//

	// Keeping position and velocity before update (useful for obstacle detection)
	agentsMovements := updateAgents(server)

	///////////////////////////////////////////////////////////////////////////
	// Collision
	///////////////////////////////////////////////////////////////////////////

	handleCollisions(server, agentsMovements, projectilesMovements)
}

func updateProjectiles(server *Server) []*collision.MovementState /*(beforeStates map[uuid.UUID]collision.CollisionMovingObjectState)*/ {

	movements := make([]*collision.MovementState, 0)

	///////////////////////////////////////////////////////////////////////////
	// On supprime les projectiles en fin de vie
	///////////////////////////////////////////////////////////////////////////

	server.state.Projectilesmutex.Lock()

	projectilesToRemove := make([]uuid.UUID, 0)
	for _, projectile := range server.state.Projectiles {
		if projectile.TTL <= 0 {
			projectilesToRemove = append(projectilesToRemove, projectile.Id)
		}
	}

	server.state.ProjectilesDeletedThisTick = make(map[uuid.UUID]*projectile.BallisticProjectile)
	for _, projectileToRemoveId := range projectilesToRemove {
		// has been set to 0 during the previous tick; pruning now (0 TTL projectiles might still have a collision later in this method)

		// Remove projectile from moving rtree
		server.state.ProjectilesDeletedThisTick[projectileToRemoveId] = server.state.Projectiles[projectileToRemoveId]

		// Remove projectile from projectiles array
		delete(server.state.Projectiles, projectileToRemoveId)
	}

	///////////////////////////////////////////////////////////////////////////
	// On conserve les états avant/après
	///////////////////////////////////////////////////////////////////////////

	for _, projectile := range server.state.Projectiles {
		beforeState := collision.CollisionMovingObjectState{
			Position: projectile.Position,
			Velocity: projectile.Velocity,
			Radius:   projectile.Radius,
		}

		projectile.Update()

		afterState := collision.CollisionMovingObjectState{
			Position: projectile.Position,
			Velocity: projectile.Velocity,
			Radius:   projectile.Radius,
		}

		bbRegion, err := collision.GetTrajectoryBoundingBox(
			beforeState.Position, beforeState.Radius,
			afterState.Position, afterState.Radius,
		)
		if err != nil {
			utils.Debug("arena-server-updatestate", "Error in updateProjectiles: could not define trajectory bbRegion")
			continue
		}

		movements = append(movements, &collision.MovementState{
			Type:           state.GeometryObjectType.Projectile,
			ID:             projectile.Id.String(),
			Before:         beforeState,
			After:          afterState,
			Rect:           bbRegion,
			AgentEmitterID: projectile.AgentEmitterId.String(),
		})
	}

	server.state.Projectilesmutex.Unlock()

	return movements
}

func updateAgents(server *Server) []*collision.MovementState {

	movements := make([]*collision.MovementState, 0)

	for _, agent := range server.agents {

		id := agent.GetId()
		beforeFullState := server.state.GetAgentState(id)
		beforeState := collision.CollisionMovingObjectState{
			Position: beforeFullState.Position,
			Velocity: beforeFullState.Velocity,
			Radius:   beforeFullState.Radius,
		}

		afterFullState := beforeFullState.Update()

		afterState := collision.CollisionMovingObjectState{
			Position: afterFullState.Position,
			Velocity: afterFullState.Velocity,
			Radius:   afterFullState.Radius,
		}

		bbRegion, err := collision.GetTrajectoryBoundingBox(
			beforeState.Position, beforeState.Radius,
			afterState.Position, afterState.Radius,
		)
		if err != nil {
			utils.Debug("arena-server-updatestate", "Error in updateAgents: could not define trajectory bbRegion")
			continue
		}

		movements = append(movements, &collision.MovementState{
			Type:   state.GeometryObjectType.Agent,
			ID:     id.String(),
			Before: beforeState,
			After:  afterState,
			Rect:   bbRegion,
		})

		server.state.SetAgentState(
			id,
			afterFullState,
		)
	}

	return movements
}
