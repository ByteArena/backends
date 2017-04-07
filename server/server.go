package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/bitly/go-notify"
	"github.com/netgusto/bytearena/server/agent"
	"github.com/netgusto/bytearena/server/comm"
	"github.com/netgusto/bytearena/server/container"
	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/server/statemutation"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

const debug = false

type Server struct {
	agents                map[uuid.UUID]agent.Agent
	agentsmutex           *sync.Mutex
	agentdir              string
	host                  string
	port                  int
	state                 *state.ServerState
	nbexpectedagents      int
	stopticking           chan bool
	commserver            *comm.CommServer
	stateobservers        []chan state.ServerState
	containerorchestrator container.ContainerOrchestrator
	tickduration          time.Duration
	tickspersec           int
	currentturn           utils.Tickturn
	currentturnmutex      *sync.Mutex
	nbhandshaked          int
}

func NewServer(host string, port int, agentdir string, nbexpectedagents int, tickspersec int, stopticking chan bool) *Server {

	orch := container.MakeContainerOrchestrator()

	return &Server{
		agents:                make(map[uuid.UUID]agent.Agent),
		agentsmutex:           &sync.Mutex{},
		agentdir:              agentdir,
		host:                  host,
		port:                  port,
		state:                 state.NewServerState(),
		nbexpectedagents:      nbexpectedagents,
		stopticking:           stopticking,
		commserver:            nil,
		containerorchestrator: orch,
		tickduration:          time.Duration((1000 / time.Duration(tickspersec)) * time.Millisecond),
		tickspersec:           tickspersec,
		currentturnmutex:      &sync.Mutex{},
	}
}

func (server *Server) Spawnagent() {

	agent := agent.MakeNetAgentImp()
	agentstate := state.MakeAgentState()

	server.setAgent(agent)
	server.state.SetAgentState(agent.GetId(), agentstate)

	container, err := server.containerorchestrator.CreateAgentContainer(agent.GetId(), server.host, server.port, server.agentdir)
	if err != nil {
		log.Panicln(err)
	}

	err = server.containerorchestrator.StartAgentContainer(container)
	if err != nil {
		log.Panicln(err)
	}

	err = server.containerorchestrator.Wait(container)
	if err != nil {
		log.Panicln(err)
	}
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

func (s *Server) GetExpectedTurn() utils.Tickturn {
	s.currentturnmutex.Lock()
	res := s.currentturn
	s.currentturnmutex.Unlock()
	return res
}

func (server *Server) Listen() {

	server.commserver = comm.NewCommServer(server.host+":"+strconv.Itoa(server.port), 1024) // 1024: max size of message in bytes
	log.Println("listening on " + server.host + ":" + strconv.Itoa(server.port))

	done := make(chan bool)
	go func() {
		err := server.commserver.Listen(server)
		if err != nil {
			log.Panicln(err)
		}
		done <- true
	}()
	<-done
}

func (server *Server) GetNbExpectedagents() int {
	return server.nbexpectedagents
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

	turn := server.GetExpectedTurn()

	var dolog bool
	if debug {
		dolog = true
	} else {
		dolog = (turn.GetSeq() % server.tickspersec) == 0
	}

	start := time.Now()
	timeoutduration := server.tickduration * 60 / 100

	if dolog {
		log.Println("-------------------------------------- Tick ! ----------------------------", turn)
	}

	// on met à jour l'état du serveur
	server.DoUpdate()

	offset := 0
	// On ticke chaque agent
	for _, ag := range server.agents {
		go func(server *Server, ag agent.Agent, turn utils.Tickturn, perception state.Perception, offset int) {
			perceptionjson, _ := json.Marshal(perception)
			message := []byte("{\"Method\": \"tick\", \"Arguments\": [" + strconv.Itoa(int(turn.GetSeq())) + "," + string(perceptionjson) + "]}\n")

			if netag, ok := ag.(agent.NetAgent); ok {
				offset := time.Microsecond * time.Duration(offset*100) // 0.1ms
				time.Sleep(offset)
				if debug {
					fmt.Print(chalk.Cyan)
					log.Println("TICKING " + turn.String() + " for " + netag.String() + " with " + offset.String() + " offset")
				}
				server.commserver.Send(message, netag.GetAddr())
			}
		}(server, ag, turn, ag.GetPerception(server.GetState()), offset)
		offset++
	}

	// On attend la réponse de chaque client, jusqu'au timeout
	wg := &sync.WaitGroup{}
	wg.Add(len(server.agents))

	nbtimedout := 0
	nbticked := 0

	for _, ag := range server.agents {
		go func(agent agent.Agent, turn utils.Tickturn) {

			myEventChan := make(chan interface{})
			notify.Start("agent:"+agent.GetId().String()+":tickedturn:"+strconv.Itoa(turn.GetSeq()), myEventChan)

			if utils.ChanTimeout(myEventChan, timeoutduration) {
				nbticked++
				//log.Print(chalk.Green)
				//log.Println("AGENT "+agent.String()+" ON TIME", chalk.Reset)
			} else {
				nbtimedout++
				log.Print(chalk.Magenta)
				log.Println("AGENT "+agent.String()+" TIMED OUT "+turn.String(), chalk.Reset)
			}

			wg.Done()
		}(ag, turn)
	}

	wg.Wait()

	// Once turn is over, immediately switch to next turn (effectively blocs late messages)
	server.SetExpectedTurn(turn.Next())

	took := time.Now().Sub(start)

	now := time.Now()
	nexttick := now.Add(server.tickduration).Add(took * -1)
	beforemutations := now

	server.ProcessMutations()

	if dolog {
		aftermutations := time.Now()
		processtook := utils.DiffMs(aftermutations, beforemutations)
		nexttickin := utils.DiffMs(nexttick, aftermutations)

		log.Print(chalk.Blue)
		log.Println(turn.String() + " OVER in " + utils.FloatToStr(utils.DurationMs(took)) + " ms;" + strconv.Itoa(nbtimedout) + " TIMED OUT AND " + strconv.Itoa(nbticked) + " IN TIME")
		log.Println("ProcessMutations() for " + turn.String() + " took " + utils.FloatToStr(processtook) + " ms; next tick in " + utils.FloatToStr(nexttickin) + " ms")
		log.Print(chalk.Reset)

		// Debug : Nombre de goroutines
		log.Print(chalk.Yellow)
		log.Println("# Nombre de goroutines en vol : " + strconv.Itoa(runtime.NumGoroutine()))
		log.Print(chalk.Reset)
	}
}

func (server *Server) DispatchAgentMessage(msg protocol.MessageWrapper) {

	ag, err := server.DoFindAgent(msg.GetAgentId().String())
	if err != nil {
		log.Panicln("Handshake : agentid does not match any known agent !")
	}

	switch msg.GetType() {
	case "Handshake":
		{
			var handshake protocol.MessageHandshakeImp
			err = json.Unmarshal(msg.GetPayload(), &handshake)
			if err != nil {
				log.Panicln(err)
			}

			ag, ok := ag.(agent.NetAgent)
			if !ok {
				log.Panicln(err)
			}

			server.setAgent(ag.SetAddr(msg.GetEmitterAddr()))

			// Handshake successful ! Matching agent is found and bound to TCPClient
			log.Println("Received handshake from agent " + ag.String() + "; agent said \"" + handshake.GetGreetings() + "\"")

			server.nbhandshaked++

			if server.nbhandshaked == server.GetNbExpectedagents() {
				server.OnAgentsReady()
			}

			break
		}
	case "Mutation":
		{
			var mutations protocol.MessageMutationsImp
			err = json.Unmarshal(msg.GetPayload(), &mutations)
			if err != nil {
				log.Println(string(msg.GetPayload()))
				log.Panicln(err)
			}

			//log.Println(string(msg.GetPayload()))

			expectedturn := server.GetExpectedTurn()
			if mutations.GetTickTurnSeq() != int(expectedturn.GetSeq()) {
				fmt.Print(chalk.Red)
				log.Println("LATE FRAME !! FROM AGENT "+msg.GetAgentId().String()+" for tick "+strconv.Itoa(int(mutations.GetTickTurnSeq()))+"; expected "+expectedturn.String(), chalk.Reset)
			} else {
				if debug {
					log.Println("GOT ANSWER FROM ", msg.GetAgentId(), "TURN", mutations.GetTickTurnSeq())
				}
			}

			mutationbatch := statemutation.StateMutationBatch{
				Turn:      expectedturn,
				AgentId:   ag.GetId(),
				Mutations: mutations.GetMutations(),
			}

			server.DoPushMutationBatch(mutationbatch)

			//ag.GetTickedChan() <- expectedtur
			notify.PostTimeout("agent:"+ag.GetId().String()+":tickedturn:"+strconv.Itoa(expectedturn.GetSeq()), nil, time.Microsecond*100)

			break
		}
	default:
		{
			log.Print(chalk.Red)
			log.Println("Unknown message type", msg)
		}
	}
}

func (server *Server) OnAgentsReady() {
	log.Print(chalk.Green)
	log.Println("All agents ready; starting in 3 seconds")
	log.Print(chalk.Reset)
	time.Sleep(time.Duration(3 * time.Second))

	server.StartTicking()
}

func (server *Server) StartTicking() {

	tickduration := server.tickduration
	stopticking := server.stopticking

	go func(server *Server) {

		log.Println("Start ticking")
		ticker := time.Tick(tickduration)

		for {
			select {
			case <-stopticking:
				{
					log.Println("Stop Ticking !")
					return
				}
			case <-ticker:
				{
					server.DoTick()
				}
			}
		}
	}(server)
}

func (server *Server) DoPushMutationBatch(batch statemutation.StateMutationBatch) {
	server.state.PushMutationBatch(batch)
}

func (server *Server) ProcessMutations() {
	server.state.ProcessMutations()
}

func (server *Server) DoUpdate() {

	// Updates physiques, liées au temps qui passe
	// Avant de récuperer les mutations de chaque tour, et même avant deconstituer la perception de chaque agent

	turn := server.GetExpectedTurn()

	// update attractor
	centerx, centery := server.state.PinCenter.Get()
	radius := 120.0

	x := centerx + radius*math.Cos(float64(turn.GetSeq())/10.0)
	y := centery + radius*math.Sin(float64(turn.GetSeq())/10.0)

	server.state.Pin = utils.MakeVector2(x, y)

	server.state.Projectilesmutex.Lock()
	for k, state := range server.state.Projectiles {

		if state.Ttl <= 0 {
			delete(server.state.Projectiles, k)
		} else {
			state.Ttl -= 1
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
