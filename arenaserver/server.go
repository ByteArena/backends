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
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
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

	agenthandshakes map[uuid.UUID]struct{}

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
		state:                 state.NewServerState(arena.GetMapContainer()),
		commserver:            nil, // initialized in Listen()
		nbhandshaked:          0,
		currentturnmutex:      &sync.Mutex{},
		mqClient:              mqClient,

		agentimages: make(map[uuid.UUID]string),

		agenthandshakes: make(map[uuid.UUID]struct{}),

		arena: arena,
	}

	return s
}

func (server *Server) GetTicksPerSecond() int {
	return server.tickspersec
}

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

		err = server.containerorchestrator.StartAgentContainer(container)

		if err != nil {
			return errors.New("Failed to start docker container for " + agent.String() + ": " + err.Error())
		}
	}

	return nil
}

func (server *Server) RegisterAgent(agentimage, agentname string) {
	arenamap := server.arena.GetMapContainer()
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

	///////////////////////////////////////////////////////////////////////////
	// Updating world state
	///////////////////////////////////////////////////////////////////////////
	server.Update()

	///////////////////////////////////////////////////////////////////////////
	// Refreshing perception for every agent
	///////////////////////////////////////////////////////////////////////////
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

	// proto := msg.GetEmitterConn().LocalAddr().Network()
	// ip := strings.Split(msg.GetEmitterConn().RemoteAddr().String(), ":")[0]
	// if proto != "tcp" || ip != "TODO:take from agent container struct"
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

	// TODO: handle monitoring stop on app:stopticking
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

func (server *Server) Start() (chan interface{}, error) {

	utils.Debug("arena", "Listen")
	block := server.Listen()

	utils.Debug("arena", "Spawn agents")
	err := server.spawnAgents()

	if err != nil {
		return nil, errors.New("Failed to spawn agents: " + err.Error())
	}

	return block, nil
}

func (server *Server) Stop() {
	log.Println("TearDown from stop")
	close(server.stopticking)

	server.TearDown()

	utils.Debug("arena", "Publish game state (stopped)")

	server.mqClient.Publish("game", "stopped", ArenaStopMessage{
		Payload: ArenaStopMessagePayload{
			ArenaServerId: server.UUID,
		},
	})

	log.Println("Close ticking")
}

func (server *Server) ProcessMutations() {
	server.DebugNbMutations++
	server.state.ProcessMutations()
}

type movingObjectTemporaryState struct {
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

	beforeStateProjectiles := updateProjectiles(server)

	//
	// Updating agents
	//

	// Keeping position and velocity before update (useful for obstacle detection)
	beforeStateAgents := updateAgents(server)

	///////////////////////////////////////////////////////////////////////////
	// Collision checks
	///////////////////////////////////////////////////////////////////////////

	// TODO: parallelism with goroutines where possible

	//
	// A: Agent/Obstacle
	//

	processAgentObstacleCollisions(server, beforeStateAgents)
	processProjectileObstacleCollisions(server, beforeStateProjectiles)

	// TODO: check for collisions:
	// * agent / agent
	// * agent / obstacle
	// * agent / projectile
	// * projectile / projectile
	// * projectile / obstacle
}

func updateProjectiles(server *Server) (beforeStates map[uuid.UUID]movingObjectTemporaryState) {

	server.state.Projectilesmutex.Lock()

	projectilesToRemove := make([]uuid.UUID, 0)
	for _, projectile := range server.state.Projectiles {
		if projectile.TTL <= 0 {
			projectilesToRemove = append(projectilesToRemove, projectile.Id)
		}
	}

	for _, projectileToRemoveId := range projectilesToRemove {
		// has been set to 0 during the previous tick; pruning now (0 TTL projectiles might still have a collision later in this method)
		// Remove projectile from projectiles array
		delete(server.state.Projectiles, projectileToRemoveId)
	}

	before := make(map[uuid.UUID]movingObjectTemporaryState)
	for _, projectile := range server.state.Projectiles {
		before[projectile.Id] = movingObjectTemporaryState{
			Position: projectile.Position,
			Velocity: projectile.Velocity,
			Radius:   projectile.Radius,
		}
	}

	for _, projectile := range server.state.Projectiles {
		projectile.Update()
	}

	server.state.Projectilesmutex.Unlock()

	return before
}

func updateAgents(server *Server) (beforeStates map[uuid.UUID]movingObjectTemporaryState) {

	before := make(map[uuid.UUID]movingObjectTemporaryState)

	for _, agent := range server.agents {
		id := agent.GetId()
		agstate := server.state.GetAgentState(id)
		before[id] = movingObjectTemporaryState{
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

	return before
}

func processProjectileObstacleCollisions(server *Server, before map[uuid.UUID]movingObjectTemporaryState) {
	for projectileid, beforestate := range before {
		projectile := server.state.GetProjectile(projectileid)

		afterstate := movingObjectTemporaryState{
			Position: projectile.Position,
			Velocity: projectile.Velocity,
			Radius:   projectile.Radius,
		}

		processMovingObjectObstacleCollision(server, beforestate, afterstate, []int{state.GeometryObjectType.ObstacleGround}, func(collisionPoint vector.Vector2) {
			//log.Println("U blocked, projectile")

			projectile.Position = collisionPoint
			projectile.Velocity = vector.MakeNullVector2()
			server.state.SetProjectile(
				projectileid,
				projectile,
			)
		})
	}
}

func processAgentObstacleCollisions(server *Server, before map[uuid.UUID]movingObjectTemporaryState) {

	for agentid, beforestate := range before {
		agentstate := server.state.GetAgentState(agentid)

		afterstate := movingObjectTemporaryState{
			Position: agentstate.Position,
			Velocity: agentstate.Velocity,
			Radius:   agentstate.Radius,
		}

		processMovingObjectObstacleCollision(server, beforestate, afterstate, nil, func(collisionPoint vector.Vector2) {
			//log.Println("U blocked, mothafucka")

			agentstate.Position = collisionPoint
			agentstate.Velocity = vector.MakeVector2(0.01, 0.01)
			server.state.SetAgentState(
				agentid,
				agentstate,
			)
		})

		// if !isInsideSurface(server, agentstate.Position) {
		// 	log.Println("HE IS OUTSIDE !!!!!!!!!")
		// 	agentstate.Position = beforestate.Position
		// 	server.state.SetAgentState(
		// 		agentid,
		// 		agentstate,
		// 	)
		// } else {
		// 	log.Println("HE IS NOT OUTSIDE !!!!!!!!!")
		// }
	}
}

func arrayContainsGeotype(needle int, haystack []int) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func processMovingObjectObstacleCollision(server *Server, beforeState, afterState movingObjectTemporaryState, geotypesIgnored []int, collisionhandler func(collision vector.Vector2)) {

	bbBeforeA, bbBeforeB := GetAgentBoundingBox(beforeState.Position, beforeState.Radius)
	bbAfterA, bbAfterB := GetAgentBoundingBox(afterState.Position, afterState.Radius)

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

		// We determine the surface occupied by the object on it's path
		// * Corresponds to a "pill", where the two ends are the bounding circles occupied by the agents (position before the move and position after the move)
		// * And the surface in between is defined the lines between the left and the right tangents of these circles
		//
		// * We then have to test collisions with the end circle
		//

		centerEdge := vector.MakeSegment2(beforeState.Position, afterState.Position)
		beforeDiameterSegment := centerEdge.OrthogonalToACentered().SetLengthFromCenter(beforeState.Radius * 2)
		afterDiameterSegment := centerEdge.OrthogonalToBCentered().SetLengthFromCenter(afterState.Radius * 2)

		beforeDiameterSegmentLeftPoint, beforeDiameterSegmentRightPoint := beforeDiameterSegment.Get()
		afterDiameterSegmentLeftPoint, afterDiameterSegmentRightPoint := afterDiameterSegment.Get()

		leftEdge := vector.MakeSegment2(beforeDiameterSegmentLeftPoint, afterDiameterSegmentLeftPoint)
		rightEdge := vector.MakeSegment2(beforeDiameterSegmentRightPoint, afterDiameterSegmentRightPoint)

		edgesToTest := []vector.Segment2{
			leftEdge,
			centerEdge,
			rightEdge,
		}

		type Collision struct {
			Point    vector.Vector2
			Obstacle *state.GeometryObject
		}

		collisions := make([]Collision, 0)

		for _, matchingObstacle := range matchingObstacles {
			geoObject := matchingObstacle.(*state.GeometryObject)
			if geotypesIgnored != nil && arrayContainsGeotype(geoObject.Type, geotypesIgnored) {
				continue
			}

			circleCollisions := trigo.LineCircleIntersectionPoints(
				geoObject.PointA,
				geoObject.PointB,
				afterState.Position,
				afterState.Radius,
			)

			for _, circleCollision := range circleCollisions {
				collisions = append(collisions, Collision{
					Point:    circleCollision,
					Obstacle: geoObject,
				})
			}

			for _, edge := range edgesToTest {
				point1, point2 := edge.Get()
				if collisionPoint, intersects, colinear, _ := trigo.IntersectionWithLineSegment(
					geoObject.PointA,
					geoObject.PointB,
					point1,
					point2,
				); intersects && !colinear {
					collisions = append(collisions, Collision{
						Point:    collisionPoint,
						Obstacle: geoObject,
					})
				}
			}
		}

		if len(collisions) > 0 {

			//normal := vector.MakeNullVector2()
			minDist := -1.0
			for _, collision := range collisions {
				thisDist := collision.Point.Sub(beforeState.Position).Mag()
				if minDist < 0 || minDist > thisDist {
					minDist = thisDist
					//normal = collision.Obstacle.Normal
				}
			}

			//normal = normal.Normalize()

			backoffDistance := beforeState.Radius + 0.001
			//nextPoint := centerEdge.Vector2().SetMag(maxDist).Sub(normal.SetMag(backoffDistance)).Add(beforeState.Position)
			nextPoint := centerEdge.Vector2().SetMag(minDist - backoffDistance).Add(beforeState.Position)

			if !isInsideGroundSurface(server, nextPoint) {

				// backtracking position to last not outside
				backsteps := 10
				railRel := afterState.Position.Sub(beforeState.Position)
				for k := 1; k <= backsteps; k++ {
					nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
					if isInsideGroundSurface(server, nextPointRel.Add(beforeState.Position)) {
						collisionhandler(nextPointRel.Add(beforeState.Position))
						return
					}
				}

				//log.Println("NOPE, BEFORESTATE GROUND !")
				collisionhandler(beforeState.Position)

			} else {
				if isInsideCollisionMesh(server, nextPoint) {

					// backtracking position to last not in obstacle
					backsteps := 30
					railRel := afterState.Position.Sub(beforeState.Position)
					for k := 1; k <= backsteps; k++ {
						nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
						if !isInsideCollisionMesh(server, nextPointRel.Add(beforeState.Position)) {
							collisionhandler(nextPointRel.Add(beforeState.Position))
							return
						}
					}

					//log.Println("NOPE, BEFORESTATE OBSTACLE !")

					collisionhandler(beforeState.Position)
				} else {
					collisionhandler(nextPoint)
				}
			}

		}
	}
}

func isInsideGroundSurface(server *Server, point vector.Vector2) bool {

	px, py := point.Get()

	bb, _ := rtreego.NewRect([]float64{px - 0.005, py - 0.005}, []float64{0.01, 0.01})
	matchingTriangles := server.state.MapMemoization.RtreeSurface.SearchIntersect(bb)

	if len(matchingTriangles) == 0 {
		return false
	}

	// On vérifie que le point est bien dans un des triangles
	for _, spatial := range matchingTriangles {
		triangle := spatial.(*state.TriangleRtreeWrapper)
		if trigo.PointIsInTriangle(point, triangle.Points[0], triangle.Points[1], triangle.Points[2]) {
			return true
		}
	}

	return false
}

func isInsideCollisionMesh(server *Server, point vector.Vector2) bool {

	px, py := point.Get()

	bb, _ := rtreego.NewRect([]float64{px - 0.005, py - 0.005}, []float64{0.01, 0.01})
	matchingTriangles := server.state.MapMemoization.RtreeCollisions.SearchIntersect(bb)

	if len(matchingTriangles) == 0 {
		return false
	}

	// On vérifie que le point est bien dans un des triangles
	for _, spatial := range matchingTriangles {
		triangle := spatial.(*state.TriangleRtreeWrapper)
		if trigo.PointIsInTriangle(point, triangle.Points[0], triangle.Points[1], triangle.Points[2]) {
			return true
		}
	}

	return false
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
