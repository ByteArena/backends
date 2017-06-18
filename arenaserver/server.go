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
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

const debug = false

type Server struct {
	host                  string
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

	agentimages map[uuid.UUID]string

	arena ArenaInstance
}

func NewServer(host string, port int, arena ArenaInstance) *Server {

	orch := container.MakeContainerOrchestrator()

	gamehost := host

	if host == "" {
		host, err := orch.GetHost()
		utils.Check(err, "Could not determine host !")

		gamehost = host
	}
	
	log.Println("Use host: "+host)

	s := &Server{
		host:                  gamehost,
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

		agentimages: make(map[uuid.UUID]string),

		arena: arena,
	}

	arena.Setup(s)

	return s
}

func (server *Server) GetTicksPerSecond() int {
	return server.tickspersec
}

func (server *Server) spawnAgents() {
	for _, ag := range server.agents {
		agentstate := server.state.GetAgentState(ag.GetId())
		agentimage := server.agentimages[ag.GetId()]

		go func(agent agent.Agent, agentstate state.AgentState, dockerimage string) {

			container, err := server.containerorchestrator.CreateAgentContainer(agent.GetId(), server.host, server.port, dockerimage)
			utils.Check(err, "Failed to create docker container for "+agent.String())

			err = server.containerorchestrator.StartAgentContainer(container)
			utils.Check(err, "Failed to start docker container for "+agent.String())

			err = server.containerorchestrator.LogsToStdOut(container)
			utils.Check(err, "Failed to follow docker container logs for "+agent.String())

			err = server.containerorchestrator.Wait(container)
			utils.Check(err, "Failed to wait docker container completion for "+agent.String())
		}(ag, agentstate, agentimage)
	}
}

func (server *Server) RegisterAgent(agentimage string) {

	agent := agent.MakeNetAgentImp()
	agentstate := state.MakeAgentState()

	server.setAgent(agent)
	server.state.SetAgentState(agent.GetId(), agentstate)

	server.agentimages[agent.GetId()] = agentimage
}

func (server *Server) SetObstacle(obstacle state.Obstacle) {
	server.state.SetObstacle(obstacle)
}

func (server *Server) setAgent(agent agent.Agent) {
	server.agentsmutex.Lock()
	server.agents[agent.GetId()] = agent
	server.agentsmutex.Unlock()
}

func (s *Server) SetExpectedTurn(turn utils.Tickturn) {
	s.currentturnmutex.Lock()
	s.currentturn = turn
	s.currentturnmutex.Unlock()
}

func (s *Server) GetTurn() utils.Tickturn {
	s.currentturnmutex.Lock()
	res := s.currentturn
	s.currentturnmutex.Unlock()
	return res
}

func (server *Server) Listen() chan interface{} {
	serveraddress := "0.0.0.0:" + strconv.Itoa(server.port)
	server.commserver = comm.NewCommServer(serveraddress, 1024) // 1024: max size of message in bytes
	log.Println("Server listening on port " + strconv.Itoa(server.port))

	if server.GetNbExpectedagents() > 0 {
		go func() {
			err := server.commserver.Listen(server)
			utils.Check(err, "Failed to listen on "+serveraddress)
			notify.Post("app:stopticking", nil)
		}()
	} else {
		server.OnAgentsReady()
	}

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
	log.Println("server::Teardown()")
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
	server.SetExpectedTurn(turn.Next())

	var dolog bool
	if debug {
		dolog = true
	} else {
		dolog = (turn.GetSeq() % server.tickspersec) == 0
	}

	if dolog {
		fmt.Print(chalk.Yellow)
		log.Println("######## Tick #####", turn, chalk.Reset)
	}

	// on met à jour l'état du serveur
	// TODO: bon moment ?
	server.DoUpdate()

	// Refreshing perception for every agent
	server.GetState().DebugIntersects = make([]vector.Vector2, 0)
	server.GetState().DebugIntersectsRejected = make([]vector.Vector2, 0)
	server.GetState().DebugPoints = make([]vector.Vector2, 0)

	for _, ag := range server.agents {
		go func(server *Server, ag agent.Agent, serverstate *state.ServerState) {

			if debug {
				fmt.Print(chalk.Cyan)
				log.Println("REFRESHING perception for " + ag.String())
			}

			agentstate := serverstate.GetAgentState(ag.GetId()) // TODO: retirer ceci; utile uniquement pour le prototypage de l'attracteur agent

			ag.SetPerception(ag.GetPerception(serverstate), server, agentstate)

		}(server, ag, server.GetState())
	}

	if dolog {
		// Debug : Nombre de goroutines
		fmt.Print(chalk.Blue)
		log.Println("# Nombre de goroutines en vol : "+strconv.Itoa(runtime.NumGoroutine()), chalk.Reset)
	}
}

/* <implementing protocol.AgentCommunicator> */

func (server *Server) NetSend(message []byte, addr net.Addr) {
	server.commserver.Send(message, addr)
}

func (server *Server) PushMutationBatch(batch protocol.StateMutationBatch) {
	server.state.PushMutationBatch(batch)
	server.ProcessMutations()
}

/* </implementing protocol.AgentCommunicator> */

func (server *Server) DispatchAgentMessage(msg protocol.MessageWrapper) {

	ag, err := server.DoFindAgent(msg.GetAgentId().String())
	utils.Check(err, "agentid does not match any known agent in received agent message !")

	switch msg.GetType() {
	case "Handshake":
		{
			var handshake protocol.MessageHandshakeImp
			err = json.Unmarshal(msg.GetPayload(), &handshake)
			utils.Check(err, "Failed to unmarshal JSON agent handshake payload")

			ag, ok := ag.(agent.NetAgent)
			utils.Assert(ok, "Failed to cast agent to NetAgent during handshake for "+ag.String())

			server.setAgent(ag.SetAddr(msg.GetEmitterAddr()))

			log.Println("Received handshake from agent " + ag.String() + "; agent said \"" + handshake.GetGreetings() + "\"")

			server.nbhandshaked++

			if server.nbhandshaked == server.GetNbExpectedagents() {
				server.OnAgentsReady()
			}

			// TODO: handle some timeout here if all agents fail to handshake

			break
		}
	case "Mutation":
		{
			var mutations protocol.MessageMutationsImp
			err = json.Unmarshal(msg.GetPayload(), &mutations)
			utils.Check(err, "Failed to unmarshal JSON agent mutation payload for agent "+ag.String()+"; "+string(msg.GetPayload()))

			turn := server.GetTurn()
			if debug {
				log.Println("GOT MUTATION FROM ", msg.GetAgentId(), "TURN", turn)
			}

			mutationbatch := protocol.StateMutationBatch{
				AgentId:   ag.GetId(),
				Mutations: mutations.GetMutations(),
			}

			server.PushMutationBatch(mutationbatch)

			notify.PostTimeout("agent:"+ag.GetId().String()+":tickedturn:"+strconv.Itoa(turn.GetSeq()), nil, time.Microsecond*100)

			break
		}
	default:
		{
			log.Print(chalk.Red)
			log.Println("Unknown message type", msg)
		}
	}
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
	log.Print(chalk.Green)
	log.Println("All agents ready; starting in .5 second")
	log.Print(chalk.Reset)
	time.Sleep(time.Duration(time.Millisecond * 500))

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
					return // exiting goroutine,
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
	log.Println("START !")
	server.spawnAgents()
	block := server.Listen()
	return block

}

func (server *Server) Stop() {
	close(server.stopticking)
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

func (server *Server) GetArena() ArenaInstance {
	return server.arena
}
