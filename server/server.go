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

func (swarm *Server) Spawnagent() {
	agent := agent.MakeAgent( /*swarm.ctx, swarm.cli, swarm.port, swarm.host, swarm.agentdir*/ )
	swarm.agents[agent.Id] = agent

	agentstate := state.MakeAgentState()
	agentstate.Radius = 8.0
	swarm.state.Agents[agent.Id] = agentstate

	container, err := swarm.containerorchestrator.CreateAgentContainer(agent.Id, swarm.host, swarm.port, swarm.agentdir)
	if err != nil {
		log.Panicln(err)
	}

	err = swarm.containerorchestrator.StartAgentContainer(container)
	if err != nil {
		log.Panicln(err)
	}

	err = swarm.containerorchestrator.Wait(container)
	if err != nil {
		log.Panicln(err)
	}
}

func (swarm *Server) Listen() {
	swarm.tcpserver = comm.NewTCPServer("tcp4", swarm.host+":"+strconv.Itoa(swarm.port), swarm)
	log.Println("listening on " + swarm.host + ":" + strconv.Itoa(swarm.port))

	done := make(chan bool)
	go func() {
		err := swarm.tcpserver.Listen()
		if err != nil {
			log.Panicln(err)
		}
		done <- true
	}()
	<-done
}

func (swarm *Server) GetNbExpectedagents() int {
	return swarm.nbexpectedagents
}

func (swarm *Server) GetState() *state.ServerState {
	return swarm.state
}

func (swarm *Server) TearDown() {
	log.Println("Swarm::Teardown()")
	swarm.containerorchestrator.TearDownAll()
}

func (swarm *Server) DoFindAgent(agentid string) agent.Agent {
	foundkey, err := uuid.FromString(agentid)
	if err != nil {
		log.Panicln(err)
	}

	agent := swarm.agents[foundkey]

	return agent
}

func (swarm *Server) OnNewClient(c *comm.TCPClient) {

}

func (swarm *Server) OnClientConnectionClosed(c *comm.TCPClient, err error) {

}

func (swarm *Server) OnAgentsReady() {
	log.Print(chalk.Green)
	log.Println("All agents ready; starting in 3 seconds")
	log.Print(chalk.Reset)
	time.Sleep(time.Duration(3 * time.Second))

	lasttick := time.Now()
	tickduration := time.Duration((1000 / time.Duration(swarm.tickspersec)) * time.Millisecond)

	ftostr := func(f float64) string {
		return strconv.FormatFloat(f, 'f', 2, 64)
	}

	diffms := func(b time.Time, a time.Time) float64 {
		return float64(b.UnixNano()-a.UnixNano()) / 1000000.0
	}

	durationms := func(d time.Duration) float64 {
		return float64(d.Nanoseconds()) / 1000000.0
	}

	swarm.tcpserver.StartTicking(tickduration, swarm.stopticking, func(took time.Duration) {

		now := time.Now()
		nexttick := now.Add(tickduration).Add(took * -1)
		sincelasttickms := diffms(now, lasttick)
		lasttick = now

		swarm.ProcessMutations()

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

func (swarm *Server) OnProcedureCall(c *comm.TCPClient, method string, arguments []interface{}) ([]interface{}, error) {

	/*
		switch method {
		case "getGreetings":
			{
				var res []interface{}
				res = append(res, swarm.getGreetings(arguments[0].(string)))
				return res, nil
			}
		}
	*/

	return nil, errors.New("Unrecognized Procedure")
}

func (swarm *Server) DoPushMutationBatch(batch statemutation.StateMutationBatch) {
	swarm.state.PushMutationBatch(batch)
}

func (swarm *Server) ProcessMutations() {
	swarm.state.ProcessMutations()
}

func (swarm *Server) DoUpdate(turn utils.Tickturn) {

	// Updates physiques, liées au temps qui passe
	// Avant de récuperer les mutations de chaque tour, et même avant deconstituer la perception de chaque agent

	// update attractor
	centerx, centery := swarm.state.PinCenter.Get()
	radius := 120.0

	x := centerx + radius*math.Cos(float64(turn.GetSeq())/10.0)
	y := centery + radius*math.Sin(float64(turn.GetSeq())/10.0)

	swarm.state.Pin = utils.MakeVector2(x, y)

	for k, state := range swarm.state.Projectiles {

		if state.Ttl <= 0 {
			delete(swarm.state.Projectiles, k)
		} else {
			state.Ttl -= 1
			swarm.state.Projectiles[k] = state
		}
	}

	// update agents
	for _, agent := range swarm.agents {
		agent.SetState(
			swarm.state,
			agent.GetState(swarm.state).Update(),
		)
	}

	// update visualisations
	swarmCloned := *swarm.state

	for _, subscriber := range swarm.stateobservers {
		go func(s chan state.ServerState) {
			s <- swarmCloned
		}(subscriber)
	}
}

func (swarm *Server) SubscribeStateObservation() chan state.ServerState {
	ch := make(chan state.ServerState)
	swarm.stateobservers = append(swarm.stateobservers, ch)
	return ch
}
