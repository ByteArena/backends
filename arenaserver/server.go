package arenaserver

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	notify "github.com/bitly/go-notify"
	uuid "github.com/satori/go.uuid"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	"github.com/bytearena/bytearena/arenaserver/perception"
	"github.com/bytearena/bytearena/arenaserver/projectile"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
)

const debug = false

type Server struct {
	host                  string
	arenaServerUUID       string
	port                  int
	stopticking           chan bool
	tickspersec           int
	containerorchestrator arenaservertypes.ContainerOrchestrator
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

	collisionListener *CollisionListener
}

func NewServer(
	host string,
	port int,
	orch arenaservertypes.ContainerOrchestrator,
	game GameInterface,
	arenaServerUUID string,
	mqClient mq.ClientInterface,
) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost()
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

	s.collisionListener = newCollisionListener(s)
	s.state.PhysicalWorld.SetContactListener(s.collisionListener)

	s.state.PhysicalWorld.SetContactFilter(newCollisionFilter(s))

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

	///////////////////////////////////////////////////////////////////////////
	// Building the physical body of the agent
	///////////////////////////////////////////////////////////////////////////

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Position.Set(agentSpawningPos.Point.X, agentSpawningPos.Point.Y)
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	body := server.state.PhysicalWorld.CreateBody(&bodydef)

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(0.5)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Agent, agent.GetId().String()))
	body.SetBullet(true)

	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////

	agentstate := state.MakeAgentState(agent.GetId(), agentname, body)

	body.SetLinearDamping(agentstate.DragForce * float64(server.tickspersec)) // aerodynamic drag

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

		arenaHostnameForAgents, err := server.containerorchestrator.GetHost()
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

func (server *Server) update() {

	server.debugNbUpdates++
	server.debugNbMutations++

	server.state.ProcessMutations()

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

		projectile := server.state.Projectiles[projectileToRemoveId]

		// Remove projectile from moving rtree
		server.state.ProjectilesDeletedThisTick[projectileToRemoveId] = server.state.Projectiles[projectileToRemoveId]

		server.state.PhysicalWorld.DestroyBody(projectile.PhysicalBody)

		// Remove projectile from projectiles array
		delete(server.state.Projectiles, projectileToRemoveId)
	}

	///////////////////////////////////////////////////////////////////////////
	// On met l'état des projectiles à jour
	///////////////////////////////////////////////////////////////////////////

	for _, projectile := range server.state.Projectiles {
		projectile.Update()
	}

	server.state.Projectilesmutex.Unlock()

	///////////////////////////////////////////////////////////////////////////
	// On met l'état des agents à jour
	///////////////////////////////////////////////////////////////////////////

	for _, agent := range server.agents {
		id := agent.GetId()
		agentstate := server.state.GetAgentState(id)
		agentstate = agentstate.Update()
		server.state.SetAgentState(
			id,
			agentstate,
		)
	}

	///////////////////////////////////////////////////////////////////////////
	// On simule le monde physique
	///////////////////////////////////////////////////////////////////////////

	before := time.Now()

	timeStep := 1.0 / float64(server.GetTicksPerSecond())

	server.state.PhysicalWorld.Step(
		timeStep,
		8, // velocityIterations; higher improves stability; default 8 in testbed
		3, // positionIterations; higher improve overlap resolution; default 3 in testbed
	)

	log.Println("Physical world step took ", float64(time.Now().UnixNano()-before.UnixNano())/1000000.0, "ms")

	///////////////////////////////////////////////////////////////////////////
	// On réagit aux contacts
	///////////////////////////////////////////////////////////////////////////

	for _, collision := range server.collisionListener.PopCollisions() {

		descriptorCollider, ok := collision.GetFixtureA().GetBody().GetUserData().(types.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		descriptorCollidee, ok := collision.GetFixtureB().GetBody().GetUserData().(types.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		if descriptorCollider.Type == types.PhysicalBodyDescriptorType.Projectile {
			// on impacte le collider
			projectileuuid, _ := uuid.FromString(descriptorCollider.ID)
			projectile := server.state.GetProjectile(projectileuuid)

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			projectile.TTL = 0
			projectile.PhysicalBody.SetLinearVelocity(box2d.MakeB2Vec2(0, 0))
			projectile.PhysicalBody.SetTransform(worldManifold.Points[0], projectile.PhysicalBody.GetAngle())

			server.state.SetProjectile(
				projectileuuid,
				projectile,
			)
		}

		if descriptorCollidee.Type == types.PhysicalBodyDescriptorType.Projectile {
			// on impacte le collider
			projectileuuid, _ := uuid.FromString(descriptorCollidee.ID)
			projectile := server.state.GetProjectile(projectileuuid)

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			projectile.TTL = 0
			projectile.PhysicalBody.SetLinearVelocity(box2d.MakeB2Vec2(0, 0))
			projectile.PhysicalBody.SetTransform(worldManifold.Points[0], projectile.PhysicalBody.GetAngle())

			server.state.SetProjectile(
				projectileuuid,
				projectile,
			)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
// Collision Handling
///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type CollisionFilter struct { /* implements box2d.B2World.B2ContactFilterInterface */
	server *Server
}

func (filter *CollisionFilter) ShouldCollide(fixtureA *box2d.B2Fixture, fixtureB *box2d.B2Fixture) bool {
	// Si projectile, ne pas collisionner agent émetteur
	// Si projectile, ne pas collisionner ground

	descriptorA, ok := fixtureA.GetBody().GetUserData().(types.PhysicalBodyDescriptor)
	if !ok {
		return false
	}

	descriptorB, ok := fixtureB.GetBody().GetUserData().(types.PhysicalBodyDescriptor)
	if !ok {
		return false
	}

	aIsProjectile := descriptorA.Type == types.PhysicalBodyDescriptorType.Projectile
	bIsProjectile := descriptorB.Type == types.PhysicalBodyDescriptorType.Projectile

	if !aIsProjectile && !bIsProjectile {
		return true
	}

	if aIsProjectile && bIsProjectile {
		return true
	}

	var projectile *types.PhysicalBodyDescriptor
	var other *types.PhysicalBodyDescriptor

	if aIsProjectile {
		projectile = &descriptorA
		other = &descriptorB
	} else {
		projectile = &descriptorB
		other = &descriptorA
	}

	if other.Type == types.PhysicalBodyDescriptorType.Obstacle {
		return true
	}

	if other.Type == types.PhysicalBodyDescriptorType.Ground {
		return false
	}

	if other.Type == types.PhysicalBodyDescriptorType.Agent {
		// fetch projectile
		projectileid, _ := uuid.FromString(projectile.ID)
		p := filter.server.GetState().GetProjectile(projectileid)
		return p.AgentEmitterId.String() != other.ID
	}

	return true
}

func newCollisionFilter(server *Server) *CollisionFilter {
	return &CollisionFilter{
		server: server,
	}
}

type CollisionListener struct { /* implements box2d.B2World.B2ContactListenerInterface */
	server          *Server
	collisionbuffer []box2d.B2ContactInterface
}

func (listener *CollisionListener) PopCollisions() []box2d.B2ContactInterface {
	defer func() { listener.collisionbuffer = make([]box2d.B2ContactInterface, 0) }()
	return listener.collisionbuffer
}

/// Called when two fixtures begin to touch.
func (listener *CollisionListener) BeginContact(contact box2d.B2ContactInterface) { // contact has to be backed by a pointer
	listener.collisionbuffer = append(listener.collisionbuffer, contact)
}

/// Called when two fixtures cease to touch.
func (listener *CollisionListener) EndContact(contact box2d.B2ContactInterface) { // contact has to be backed by a pointer
	//log.Println("END:COLLISION !!!!!!!!!!!!!!")
}

/// This is called after a contact is updated. This allows you to inspect a
/// contact before it goes to the solver. If you are careful, you can modify the
/// contact manifold (e.g. disable contact).
/// A copy of the old manifold is provided so that you can detect changes.
/// Note: this is called only for awake bodies.
/// Note: this is called even when the number of contact points is zero.
/// Note: this is not called for sensors.
/// Note: if you set the number of contact points to zero, you will not
/// get an EndContact callback. However, you may get a BeginContact callback
/// the next step.
func (listener *CollisionListener) PreSolve(contact box2d.B2ContactInterface, oldManifold box2d.B2Manifold) { // contact has to be backed by a pointer
	//log.Println("PRESOLVE !!!!!!!!!!!!!!")
}

/// This lets you inspect a contact after the solver is finished. This is useful
/// for inspecting impulses.
/// Note: the contact manifold does not include time of impact impulses, which can be
/// arbitrarily large if the sub-step is small. Hence the impulse is provided explicitly
/// in a separate data structure.
/// Note: this is only called for contacts that are touching, solid, and awake.
func (listener *CollisionListener) PostSolve(contact box2d.B2ContactInterface, impulse *box2d.B2ContactImpulse) { // contact has to be backed by a pointer
	//log.Println("POSTSOLVE !!!!!!!!!!!!!!")
}

func newCollisionListener(server *Server) *CollisionListener {
	return &CollisionListener{
		server: server,
	}
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
