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
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
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

	arena Game
}

func NewServer(host string, port int, orch container.ContainerOrchestrator, arena Game) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost(&orch)
		utils.Check(err, "Could not determine arena-server host/ip.")

		gamehost = host
	}

	s := &Server{
		host:                  gamehost,
		port:                  port,
		stopticking:           make(chan struct{}),
		tickspersec:           arena.GetTps(),
		containerorchestrator: orch,
		agents:                make(map[uuid.UUID]agent.Agent),
		agentsmutex:           &sync.Mutex{},
		state:                 state.NewServerState(arena.GetMapContainer()),
		commserver:            nil, // initialized in Listen()
		nbhandshaked:          0,
		currentturnmutex:      &sync.Mutex{},

		agentimages: make(map[uuid.UUID]string),

		arena: arena,
	}

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

			err = server.containerorchestrator.Wait(container)
			utils.Check(err, "Failed to wait docker container completion for "+agent.String())
		}(ag, agentstate, agentimage)
	}
}

func (server *Server) RegisterAgent(agentimage string) {
	arenamap := server.arena.GetMapContainer()
	agentSpawnPointIndex := len(server.agents)

	if agentSpawnPointIndex >= len(arenamap.Data.Starts) {
		log.Panicln("Agent cannot spawn, no starting point left")
	}

	agentSpawningPos := arenamap.Data.Starts[agentSpawnPointIndex]

	agent := agent.MakeNetAgentImp()
	agentstate := state.MakeAgentState(agent.GetId(), agentSpawningPos)

	server.setAgent(agent)
	server.state.SetAgentState(agent.GetId(), agentstate)

	server.agentimages[agent.GetId()] = agentimage
}

// func (server *Server) SetObstacle(obstacle state.Obstacle) {
// 	server.state.SetObstacle(obstacle)
// }

func (server *Server) setAgent(agent agent.Agent) {
	server.agentsmutex.Lock()
	server.agents[agent.GetId()] = agent
	server.agentsmutex.Unlock()
}

func (s *Server) SetTurn(turn utils.Tickturn) {
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
	server.commserver = comm.NewCommServer(serveraddress)
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
	server.SetTurn(turn.Next())

	dolog := (turn.GetSeq() % server.tickspersec) == 0

	if dolog {
		fmt.Print(chalk.Yellow)
		log.Println("######## Tick #####", turn, chalk.Reset)
	}

	// on met à jour l'état du serveur
	// TODO: bon moment ?
	server.Update()

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

type AgentStateKeptForObstacleCollisionDetection struct {
	Position vector.Vector2
	Velocity vector.Vector2
	Radius   float64
}

func (server *Server) Update() {

	server.DebugNbUpdates++

	///////////////////////////////////////////////////////////////////////////
	// Updates physiques, liées au temps qui passe
	// Avant de récuperer les mutations de chaque tour, et même avant de constituer la perception de chaque agent
	///////////////////////////////////////////////////////////////////////////

	//
	// Updating projectiles
	//

	server.state.Projectilesmutex.Lock()
	for i, projectile := range server.state.Projectiles {
		if projectile.TTL <= 0 {
			// has been set to 0 during the previous tick; pruning now (0 TTL projectiles might still have a collision later in this method)
			// Remove projectile from projectiles array
			server.state.Projectiles[i] = server.state.Projectiles[len(server.state.Projectiles)-1]
			server.state.Projectiles = server.state.Projectiles[:len(server.state.Projectiles)-1]
		} else {
			projectile.Update()
		}
	}
	server.state.Projectilesmutex.Unlock()

	//
	// Updating agents
	//

	// Keeping position and velocity before update (useful for obstacle detection)

	before := make(map[uuid.UUID]AgentStateKeptForObstacleCollisionDetection)

	for _, agent := range server.agents {
		id := agent.GetId()
		agstate := server.state.GetAgentState(id)
		before[id] = AgentStateKeptForObstacleCollisionDetection{
			Position: agstate.Position,
			Velocity: agstate.Velocity,
			Radius:   agstate.Radius,
		}
	}

	for _, agent := range server.agents {
		server.state.SetAgentState(
			agent.GetId(),
			server.state.GetAgentState(agent.GetId()).Update(),
		)
	}

	///////////////////////////////////////////////////////////////////////////
	// Collision checks
	///////////////////////////////////////////////////////////////////////////

	// TODO: parallelism with goroutines wher possible

	//
	// A: Agent/Obstacle
	//

	//
	// * For each before state:
	//   * Determine the bounding box enclosing the n-1 and n positions for an agent
	//   * Get all obstacles overlapping the bounding box
	//	 * For each obstacle found, determine if they were actually crossed or not
	//

	for agentid, beforestate := range before {
		afterstate := server.state.GetAgentState(agentid)

		bbBeforeA, bbBeforeB := GetAgentBoundingBox(beforestate.Position, beforestate.Radius)
		bbAfterA, bbAfterB := GetAgentBoundingBox(afterstate.Position, afterstate.Radius)

		var minX, minY *float64
		var maxX, maxY *float64

		for _, point := range []vector.Vector2{bbBeforeA, bbBeforeB, bbAfterA, bbAfterB} {

			x, y := point.Get()

			if minX == nil || x < *minX {
				minX = &(x)
			}

			if minY == nil || y < *minY {
				minY = &(y)
			}

			if maxX == nil || x > *maxX {
				maxX = &(x)
			}

			if maxY == nil || y > *maxY {
				maxY = &(y)
			}
		}

		bbRegion, err := rtreego.NewRect([]float64{*minX, *minY}, []float64{*maxX - *minX, *maxY - *minY})
		utils.Check(err, "rtreego Error")

		//start := time.Now().UnixNano()
		matchingObstacles := server.state.MapMemoization.RtreeObstacles.SearchIntersect(bbRegion)
		//fmt.Println("Took", time.Now().UnixNano()-start, "nanoseconds")

		if len(matchingObstacles) > 0 {

			// Fine collision checking

			for _, matchingObstacle := range matchingObstacles {
				geoObject := matchingObstacle.(*state.GeometryObject)

				intersections := make([]vector.Vector2, 0)

				// 1---2
				// |   |
				// 4---3

				boxPoints := []vector.Vector2{
					vector.MakeVector2(bbAfterA.GetX(), bbAfterA.GetY()),
					vector.MakeVector2(bbAfterB.GetX(), bbAfterA.GetY()),
					vector.MakeVector2(bbAfterB.GetX(), bbAfterB.GetY()),
					vector.MakeVector2(bbAfterA.GetX(), bbAfterB.GetY()),
				}

				// Check agent bounding box intersection with line A---B defined by the obstacle
				for i := 0; i < len(boxPoints); i++ {

					boxPoint1 := boxPoints[i]
					boxPoint2 := boxPoints[(i+1)%len(boxPoints)]

					if point, intersects, colinear, _ := trigo.IntersectionWithLineSegment(geoObject.PointA,
						geoObject.PointB,
						boxPoint1,
						boxPoint2,
					); intersects && !colinear {
						// INTERSECT RIGHT
						intersections = append(intersections, point)
					}
				}

				if len(intersections) > 0 {
					// Check if intersection is within collision circle

					collisions := make([]vector.Vector2, 0)

					radiusSq := afterstate.Radius * afterstate.Radius
					for _, intersection := range intersections {
						if intersection.Sub(afterstate.Position).MagSq() >= radiusSq {
							// Circle collision !
							collisions = append(collisions, intersection)
						}
					}

					if len(collisions) > 0 {
						log.Println("U blocked, mothafucka")

						newState := server.state.GetAgentState(agentid)
						newState.Position = beforestate.Position
						newState.Velocity = vector.MakeNullVector2()
						server.state.SetAgentState(
							agentid,
							newState,
						)
					}
				}
			}
		}
	}

	// TODO: check for collisions:
	// * agent / agent
	// * agent / obstacle
	// * agent / projectile
	// * projectile / projectile
	// * projectile / obstacle

	///////////////////////////////////////////////////////////////////////////
	// Pushing updated state to viz
	// TODO: is this the right place ?
	///////////////////////////////////////////////////////////////////////////

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

func GetAgentBoundingBox(center vector.Vector2, radius float64) (vector.Vector2, vector.Vector2) {
	x, y := center.Get()
	return vector.MakeVector2(x-radius, y-radius), vector.MakeVector2(x+radius, y+radius)
}
