package server

import (
	"errors"
	"log"
	"math"
	"runtime"
	"strconv"
	"time"

	"github.com/netgusto/bytearena/server/agent"
	"github.com/netgusto/bytearena/server/comm"
	"github.com/netgusto/bytearena/server/container"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/server/statemutation"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

type Server struct {
	agents                map[uuid.UUID]agent.Agent
	agentdir              string
	host                  string
	port                  int
	state                 *state.ServerState
	nbexpectedagents      int
	tickspersec           int
	stopticking           chan bool
	tcpserver             *comm.TCPServer
	stateobservers        []chan state.ServerState
	containerorchestrator container.ContainerOrchestrator
}

func NewServer(host string, port int, agentdir string, nbexpectedagents int, tickspersec int, stopticking chan bool) *Server {

	orch := container.MakeContainerOrchestrator()

	return &Server{
		agents:                make(map[uuid.UUID]agent.Agent),
		agentdir:              agentdir,
		host:                  host,
		port:                  port,
		state:                 state.NewServerState(),
		nbexpectedagents:      nbexpectedagents,
		tickspersec:           tickspersec,
		stopticking:           stopticking,
		tcpserver:             nil,
		containerorchestrator: orch,
	}
}

func (server *Server) Spawnagent() {
	agent := agent.MakeAgent()
	server.agents[agent.Id] = agent

	agentstate := state.MakeAgentState()
	agentstate.Radius = 8.0
	server.state.Agents[agent.Id] = agentstate

	container, err := server.containerorchestrator.CreateAgentContainer(agent.Id, server.host, server.port, server.agentdir)
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

func (server *Server) Listen() {
	server.tcpserver = comm.NewTCPServer("tcp4", server.host+":"+strconv.Itoa(server.port), server)
	log.Println("listening on " + server.host + ":" + strconv.Itoa(server.port))

	done := make(chan bool)
	go func() {
		err := server.tcpserver.Listen()
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

	if foundagent, ok := server.agents[foundkey]; ok {
		return foundagent, nil
	}

	return emptyagent, errors.New("Agent" + agentid + " not found")
}

func (server *Server) OnNewClient(c *comm.TCPClient) {

}

func (server *Server) OnClientConnectionClosed(c *comm.TCPClient, err error) {

}

func (server *Server) OnAgentsReady() {
	log.Print(chalk.Green)
	log.Println("All agents ready; starting in 3 seconds")
	log.Print(chalk.Reset)
	time.Sleep(time.Duration(3 * time.Second))

	lasttick := time.Now()
	tickduration := time.Duration((1000 / time.Duration(server.tickspersec)) * time.Millisecond)

	ftostr := func(f float64) string {
		return strconv.FormatFloat(f, 'f', 2, 64)
	}

	diffms := func(b time.Time, a time.Time) float64 {
		return float64(b.UnixNano()-a.UnixNano()) / 1000000.0
	}

	durationms := func(d time.Duration) float64 {
		return float64(d.Nanoseconds()) / 1000000.0
	}

	server.tcpserver.StartTicking(tickduration, server.stopticking, func(took time.Duration) {

		now := time.Now()
		nexttick := now.Add(tickduration).Add(took * -1)
		sincelasttickms := diffms(now, lasttick)
		lasttick = now

		server.ProcessMutations()

		nowaftermutations := time.Now()
		processtook := diffms(nowaftermutations, lasttick)
		nexttickin := diffms(nexttick, nowaftermutations)
		log.Print(chalk.Blue)
		log.Println("All agents ticked in " + ftostr(durationms(took)) + " ms; Since last tick " + ftostr(sincelasttickms) + " ms")
		log.Println("ProcessMutations() took " + ftostr(processtook) + " ms; next tick in " + ftostr(nexttickin) + " ms")
		log.Print(chalk.Reset)

		// Debug : Nombre de goroutines
		log.Print(chalk.Yellow)
		log.Println("# Nombre de goroutines en vol : " + strconv.Itoa(runtime.NumGoroutine()))
		log.Print(chalk.Reset)

	})
}

func (server *Server) OnProcedureCall(c *comm.TCPClient, method string, arguments []interface{}) ([]interface{}, error) {

	/*
		switch method {
		case "getGreetings":
			{
				var res []interface{}
				res = append(res, server.getGreetings(arguments[0].(string)))
				return res, nil
			}
		}
	*/

	return nil, errors.New("Unrecognized Procedure")
}

func (server *Server) DoPushMutationBatch(batch statemutation.StateMutationBatch) {
	server.state.PushMutationBatch(batch)
}

func (server *Server) ProcessMutations() {
	server.state.ProcessMutations()
}

func (server *Server) DoUpdate(turn utils.Tickturn) {

	// Updates physiques, liées au temps qui passe
	// Avant de récuperer les mutations de chaque tour, et même avant deconstituer la perception de chaque agent

	// update attractor
	centerx, centery := server.state.PinCenter.Get()
	radius := 120.0

	x := centerx + radius*math.Cos(float64(turn.GetSeq())/10.0)
	y := centery + radius*math.Sin(float64(turn.GetSeq())/10.0)

	server.state.Pin = utils.MakeVector2(x, y)

	for k, state := range server.state.Projectiles {

		if state.Ttl <= 0 {
			delete(server.state.Projectiles, k)
		} else {
			state.Ttl -= 1
			server.state.Projectiles[k] = state
		}
	}

	// update agents
	for _, agent := range server.agents {
		agent.SetState(
			server.state,
			agent.GetState(server.state).Update(),
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
