package arenaserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	"github.com/bytearena/bytearena/arenaserver/container"
	"github.com/bytearena/bytearena/arenaserver/perception"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

const debug = false

type Server struct {
	host                  string
	UUID                  string
	port                  int
	stopticking           chan struct{}
	tickspersec           int
	containerorchestrator container.ContainerOrchestrator
	agents                map[uuid.UUID]agent.Agent
	agentsmutex           *sync.Mutex
	state                 *state.ServerState
	commserver            *comm.CommServer
	nbhandshaked          int
	currentturn           utils.Tickturn
	currentturnmutex      *sync.Mutex
	stateobservers        []chan state.ServerState
	DebugNbMutations      int
	DebugNbUpdates        int
	mqClient              mq.ClientInterface

	agentimages map[uuid.UUID]string

	arena Game
}

type ArenaStopMessagePayload struct {
	ArenaServerId string `json:"arenaserverid"`
}

type ArenaStopMessage struct {
	Payload ArenaStopMessagePayload `json:"payload"`
}

func NewServer(host string, port int, orch container.ContainerOrchestrator, arena Game, UUID string, mqClient mq.ClientInterface) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost(&orch)
		utils.Check(err, "Could not determine arena-server host/ip.")

		gamehost = host
	}

	s := &Server{
		host:                  gamehost,
		UUID:                  UUID,
		port:                  port,
		stopticking:           make(chan struct{}),
		tickspersec:           arena.GetTps(),
		containerorchestrator: orch,
		agents:                make(map[uuid.UUID]agent.Agent),
		agentsmutex:           &sync.Mutex{},
		state:                 state.NewServerState(),
		commserver:            nil, // initialized in Listen()
		nbhandshaked:          0,
		currentturnmutex:      &sync.Mutex{},
		mqClient:              mqClient,

		agentimages: make(map[uuid.UUID]string),

		arena: arena,
	}

	return s
}

func (server *Server) GetTicksPerSecond() int {
	return server.tickspersec
}

// func (server *Server) spawnAgents() error {

// 	for _, agent := range server.agents {
// 		agentimage := server.agentimages[agent.GetId()]

// 		container, err := server.containerorchestrator.CreateAgentContainer(agent.GetId(), server.host, server.port, agentimage)

// 		if err != nil {
// 			return errors.New("Cannot create agent container: " + err.Error())
// 		}

// 		err = server.containerorchestrator.StartAgentContainer(container)

// 		if err != nil {
// 			return errors.New("Failed to start docker container: " + err.Error())
// 		}
// 	}

// 	return nil
// }

func (server *Server) spawnAgents() error {

	for _, ag := range server.agents {
		agentstate := server.state.GetAgentState(ag.GetId())
		agentimage := server.agentimages[ag.GetId()]

		go func(agent agent.Agent, agentstate state.AgentState, dockerimage string) {

			container, err := server.containerorchestrator.CreateAgentContainer(agent.GetId(), server.host, server.port, dockerimage)
			utils.Check(err, "Failed to create docker container for "+agent.String())

			err = server.containerorchestrator.StartAgentContainer(container)
			utils.Check(err, "Failed to start docker container for "+agent.String())

			server.containerorchestrator.Wait(container)

			utils.Debug("arena", "One container exited")
			server.TearDown()
			server.Stop()

		}(ag, agentstate, agentimage)
	}

	return nil
}

func (server *Server) RegisterAgent(agentimage string) {
	arenamap := server.arena.GetMapContainer()
	agentSpawnPointIndex := len(server.agents)

	if agentSpawnPointIndex >= len(arenamap.Data.Starts) {
		utils.Debug("arena", "Agent "+agentimage+" cannot spawn, no starting point left")
		return
	}

	agentSpawningPos := arenamap.Data.Starts[agentSpawnPointIndex]

	agent := agent.MakeNetAgentImp()
	agentstate := state.MakeAgentState(agentSpawningPos)

	server.setAgent(agent)
	server.state.SetAgentState(agent.GetId(), agentstate)

	server.agentimages[agent.GetId()] = agentimage

	utils.Debug("arena", "Registrer agent "+agentimage)
}

func (server *Server) setAgent(agent agent.Agent) {
	server.agentsmutex.Lock()
	defer server.agentsmutex.Unlock()

	server.agents[agent.GetId()] = agent
}

func (s *Server) SetTurn(turn utils.Tickturn) {
	s.currentturnmutex.Lock()
	defer s.currentturnmutex.Unlock()

	s.currentturn = turn
}

func (s *Server) GetTurn() utils.Tickturn {
	s.currentturnmutex.Lock()
	defer s.currentturnmutex.Unlock()

	res := s.currentturn

	return res
}

func (server *Server) Listen() chan interface{} {
	serveraddress := "0.0.0.0:" + strconv.Itoa(server.port)
	server.commserver = comm.NewCommServer(serveraddress)

	utils.Debug("arena", "Server listening on port "+strconv.Itoa(server.port))

	err := server.commserver.Listen(server)
	utils.Check(err, "Failed to listen on "+serveraddress)

	notify.Post("app:stopticking", nil)

	block := make(chan interface{})
	notify.Start("app:stopticking", block)

	return block
}

func (server *Server) GetNbExpectedagents() int {
	return len(server.arena.GetContestants())
}

func (server *Server) GetState() *state.ServerState {
	return server.state
}

func (server *Server) TearDown() {
	utils.Debug("arena", "teardown")
	server.containerorchestrator.TearDownAll()
}

func (server *Server) DoFindAgent(agentid string) (agent.Agent, error) {
	var emptyagent agent.Agent

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

func (server *Server) DoTick() {

	turn := server.GetTurn()
	server.SetTurn(turn.Next())

	dolog := (turn.GetSeq() % server.tickspersec) == 0

	if dolog {
		fmt.Print(chalk.Yellow)
		log.Println("######## Tick #####", turn, chalk.Reset)
	}

	// on met à jour l'état du serveur
	// TODO: bon moment ?
	server.DoUpdate()

	// Refreshing perception for every agent
	server.GetState().DebugPoints = make([]vector.Vector2, 0)

	arenamap := server.arena.GetMapContainer()

	for _, ag := range server.agents {
		go func(server *Server, ag agent.Agent, serverstate *state.ServerState, arenamap *mapcontainer.MapContainer) {

			p := perception.ComputeAgentPerception(arenamap, serverstate, ag)

			err := ag.SetPerception(p, server)
			if err != nil {
				fmt.Print(chalk.Red)
				log.Println("ERROR: could not set perception on agent", ag.GetId().String(), chalk.Reset)
			}

		}(server, ag, server.GetState(), arenamap)
	}

	if dolog {
		// Debug : Nombre de goroutines
		fmt.Print(chalk.Blue)
		log.Println("# Nombre de goroutines en vol : "+strconv.Itoa(runtime.NumGoroutine()), chalk.Reset)
	}
}

/* <implementing protocol.AgentCommunicator> */

func (server *Server) NetSend(message []byte, conn net.Conn) error {
	return server.commserver.Send(message, conn)
}

func (server *Server) PushMutationBatch(batch protocol.StateMutationBatch) {
	server.state.PushMutationBatch(batch)
	server.ProcessMutations()
}

/* </implementing protocol.AgentCommunicator> */

func (server *Server) DispatchAgentMessage(msg protocol.MessageWrapper) error {

	ag, err := server.DoFindAgent(msg.GetAgentId().String())
	if err != nil {
		return errors.New("DispatchAgentMessage: agentid does not match any known agent in received agent message !;" + msg.GetAgentId().String())
	}

	switch msg.GetType() {
	case "Handshake":
		{
			var handshake protocol.MessageHandshakeImp
			err = json.Unmarshal(msg.GetPayload(), &handshake)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent handshake payload for agent " + msg.GetAgentId().String() + "; " + string(msg.GetPayload()))
			}

			ag, ok := ag.(agent.NetAgent)
			if !ok {
				return errors.New("DispatchAgentMessage: Failed to cast agent to NetAgent during handshake for " + ag.String())
			}

			ag = ag.SetConn(msg.GetEmitterConn())
			server.setAgent(ag)

			utils.Debug("arena", "Received handshake from agent "+ag.String()+"; agent said \""+handshake.GetGreetings()+"\"")

			server.nbhandshaked++

			if server.nbhandshaked == server.GetNbExpectedagents() {
				server.OnAgentsReady()
			}

			// TODO: handle some timeout here if all agents fail to handshake

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
			log.Print(chalk.Red)
			log.Println("Unknown message type", msg)
			return errors.New("DispatchAgentMessage: Unknown message type" + msg.GetType())
		}
	}

	return nil
}

func (server *Server) monitoring() {
	monitorfreq := time.Second
	debugNbMutations := 0
	debugNbUpdates := 0
	for {
		select {
		case <-time.After(monitorfreq):
			{
				fmt.Print(chalk.Cyan)
				log.Println(
					"-- MONITORING --",
					/*server.DebugNbMutations, "mutations,", */ server.DebugNbMutations-debugNbMutations, "mutations per", monitorfreq,
					";",
					/*server.DebugNbUpdates, "updates,", */ server.DebugNbUpdates-debugNbUpdates, "updates per", monitorfreq,
					chalk.Reset,
				)

				debugNbMutations = server.DebugNbMutations
				debugNbUpdates = server.DebugNbUpdates

			}
		}
	}
}

func (server *Server) OnAgentsReady() {
	utils.Debug("arena", "Agents are ready; starting in 1 second")
	time.Sleep(time.Duration(time.Second * 1))

	go server.monitoring()

	server.startTicking()
}

func (server *Server) startTicking() {

	go func() {

		tickduration := time.Duration((1000000 / time.Duration(server.tickspersec)) * time.Microsecond)
		ticker := time.Tick(tickduration)

		for {
			select {
			case <-server.stopticking:
				{
					log.Println("Received stop ticking signal")
					notify.Post("app:stopticking", nil)
					return
				}
			case <-ticker:
				{
					server.DoTick()
				}
			}
		}
	}()
}

func (server *Server) Start() chan interface{} {
	utils.Debug("arena", "Spawn agents")
	err := server.spawnAgents()

	utils.CheckWithFunc(err, func() string {
		return "Failed to spawn agents: " + err.Error()
	})

	utils.Debug("arena", "Listen")
	block := server.Listen()

	return block

}

func (server *Server) Stop() {
	log.Println("TearDown from stop")
	server.TearDown()

	server.mqClient.Publish("game", "stopped", ArenaStopMessage{
		Payload: ArenaStopMessagePayload{
			ArenaServerId: server.UUID,
		},
	})

	close(server.stopticking)
	log.Println("Close ticking")
}

func (server *Server) ProcessMutations() {
	server.DebugNbMutations++
	server.state.ProcessMutations()
}

func (server *Server) DoUpdate() {
	server.DebugNbUpdates++

	// Updates physiques, liées au temps qui passe
	// Avant de récuperer les mutations de chaque tour, et même avant deconstituer la perception de chaque agent

	server.state.Projectilesmutex.Lock()
	for k, state := range server.state.Projectiles {

		if state.Ttl <= 0 {
			delete(server.state.Projectiles, k)
		} else {
			state.Ttl--
			server.state.Projectiles[k] = state
		}
	}
	server.state.Projectilesmutex.Unlock()

	// update agents
	for _, agent := range server.agents {
		server.state.SetAgentState(
			agent.GetId(),
			server.state.GetAgentState(agent.GetId()).Update(),
		)
	}

	// update visualisations
	serverCloned := *server.state

	for _, subscriber := range server.stateobservers {
		go func(s chan state.ServerState) {
			s <- serverCloned
		}(subscriber)
	}
}

func (server *Server) SubscribeStateObservation() chan state.ServerState {
	ch := make(chan state.ServerState)
	server.stateobservers = append(server.stateobservers, ch)
	return ch
}

func (server *Server) GetArena() Game {
	return server.arena
}
