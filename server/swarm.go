package server

import (
	"context"
	"errors"
	"log"
	"math"
	"runtime"
	"strconv"
	"time"

	"github.com/docker/docker/client"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

type Swarm struct {
	ctx              context.Context
	cli              *client.Client
	agents           map[uuid.UUID]*Agent
	agentdir         string
	host             string
	port             int
	state            *SwarmState
	nbexpectedagents int
	tickspersec      int
	stopticking      chan bool
	tcpserver        *TCPServer
	stateobservers   []chan SwarmState
}

func NewSwarm(ctx context.Context, host string, port int, agentdir string, nbexpectedagents int, tickspersec int, stopticking chan bool) *Swarm {

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Panicln(err)
	}

	/*_, err = cli.ImagePull(ctx, "docker.io/library/node", types.ImagePullOptions{})
	if err != nil {
		log.Panicln(err)
	}*/

	return &Swarm{
		ctx:              ctx,
		cli:              cli,
		agents:           make(map[uuid.UUID]*Agent),
		agentdir:         agentdir,
		host:             host,
		port:             port,
		state:            NewSwarmState(),
		nbexpectedagents: nbexpectedagents,
		tickspersec:      tickspersec,
		stopticking:      stopticking,
		tcpserver:        nil,
	}
}

func (swarm *Swarm) Spawnagent() {
	agent := NewAgent(swarm)
	swarm.agents[agent.id] = agent

	agentstate := MakeAgentState()
	agentstate.Radius = 8.0
	swarm.state.Agents[agent.id] = agentstate

	err := agent.Start()
	if err != nil {
		log.Panicln(err)
	}

	err = agent.Wait()
	if err != nil {
		log.Panicln(err)
	}
}

func (swarm *Swarm) Listen() {
	swarm.tcpserver = NewTCPServer("tcp4", swarm.host+":"+strconv.Itoa(swarm.port), swarm)
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

func (swarm *Swarm) Teardown() {
	log.Println("Swarm::Teardown()")
	for _, agent := range swarm.agents {
		agent.Teardown()
	}
}

func (swarm *Swarm) FindAgent(agentid string) (*Agent, error) {
	foundkey, err := uuid.FromString(agentid)
	if err != nil {
		return nil, err
	}

	agent := swarm.agents[foundkey]

	return agent, nil
}

func (swarm *Swarm) OnNewClient(c *TCPClient) {

}

func (swarm *Swarm) OnClientConnectionClosed(c *TCPClient, err error) {

}

func (swarm *Swarm) OnAgentsReady() {
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

func (swarm *Swarm) OnProcedureCall(c *TCPClient, method string, arguments []interface{}) ([]interface{}, error) {

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

func (swarm *Swarm) PushMutationBatch(batch StateMutationBatch) {
	swarm.state.PushMutationBatch(batch)
}

func (swarm *Swarm) ProcessMutations() {
	swarm.state.ProcessMutations()
}

func (swarm *Swarm) update(turn tickturn) {

	// Updates physiques, liées au temps qui passe
	// Avant de récuperer les mutations de chaque tour, et même avant deconstituer la perception de chaque agent

	// update attractor
	centerx, centery := swarm.state.PinCenter.Get()
	radius := 120.0

	x := centerx + radius*math.Cos(float64(turn.seq)/10.0)
	y := centery + radius*math.Sin(float64(turn.seq)/10.0)

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
		agent.SetState(agent.GetState().update())
	}

	// update visualisations
	swarmCloned := *swarm.state

	for _, subscriber := range swarm.stateobservers {
		go func(s chan SwarmState) {
			s <- swarmCloned
		}(subscriber)
	}
}

func (swarm *Swarm) SubscribeStateObservation() chan SwarmState {
	ch := make(chan SwarmState)
	swarm.stateobservers = append(swarm.stateobservers, ch)
	return ch
}
