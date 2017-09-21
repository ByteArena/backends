package state

import (
	"encoding/json"
	"math"
	"sync"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/bytearena/game/entities"
	uuid "github.com/satori/go.uuid"
)

type ServerState struct {
	Agents      map[uuid.UUID](entities.AgentState)
	Agentsmutex *sync.Mutex

	Projectiles                map[uuid.UUID](*entities.BallisticProjectile)
	Projectilesmutex           *sync.Mutex
	ProjectilesDeletedThisTick map[uuid.UUID](*entities.BallisticProjectile)

	pendingmutations []protocol.AgentMutationBatch
	mutationsmutex   *sync.Mutex

	DebugPoints      []vector.Vector2
	debugPointsMutex *sync.Mutex

	PhysicalWorld *box2d.B2World

	MapMemoization *MapMemoization
}

/* ***************************************************************************/
/* ServerState implementation */
/* ***************************************************************************/

func NewServerState(arenaMap *mapcontainer.MapContainer) *ServerState {

	return &ServerState{
		Agents:      make(map[uuid.UUID](entities.AgentState)),
		Agentsmutex: &sync.Mutex{},

		Projectiles:                make(map[uuid.UUID]*entities.BallisticProjectile),
		Projectilesmutex:           &sync.Mutex{},
		ProjectilesDeletedThisTick: make(map[uuid.UUID]*entities.BallisticProjectile),

		pendingmutations: make([]protocol.AgentMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},

		DebugPoints:      make([]vector.Vector2, 0),
		debugPointsMutex: &sync.Mutex{},
		MapMemoization:   initializeMapMemoization(arenaMap),
		PhysicalWorld:    buildPhysicalWorld(arenaMap),
	}
}

func (serverstate *ServerState) GetProjectile(projectileid uuid.UUID) *entities.BallisticProjectile {
	serverstate.Projectilesmutex.Lock()
	res := serverstate.Projectiles[projectileid]
	serverstate.Projectilesmutex.Unlock()

	return res
}

func (serverstate *ServerState) SetProjectile(projectileid uuid.UUID, projectile *entities.BallisticProjectile) {
	serverstate.Projectilesmutex.Lock()
	serverstate.Projectiles[projectileid] = projectile
	serverstate.Projectilesmutex.Unlock()
}

func (serverstate *ServerState) GetAgentState(agentid uuid.UUID) entities.AgentState {
	serverstate.Agentsmutex.Lock()
	res := serverstate.Agents[agentid]
	serverstate.Agentsmutex.Unlock()

	return res
}

func (serverstate *ServerState) SetAgentState(agentid uuid.UUID, agentstate entities.AgentState) {
	serverstate.Agentsmutex.Lock()
	serverstate.Agents[agentid] = agentstate
	serverstate.Agentsmutex.Unlock()
}

func (serverstate *ServerState) PushMutationBatch(batch protocol.AgentMutationBatch) {
	serverstate.mutationsmutex.Lock()
	serverstate.pendingmutations = append(serverstate.pendingmutations, batch)
	serverstate.mutationsmutex.Unlock()
}

func (serverstate *ServerState) ProcessMutations() {

	serverstate.mutationsmutex.Lock()
	mutations := serverstate.pendingmutations
	serverstate.pendingmutations = make([]protocol.AgentMutationBatch, 0)
	serverstate.mutationsmutex.Unlock()

	for _, batch := range mutations {

		nbmutations := 0

		serverstate.Agentsmutex.Lock()
		agentstate := serverstate.Agents[batch.AgentId]
		newstate := agentstate.Clone()
		serverstate.Agentsmutex.Unlock()

		// Ordering actions
		// This is important because operations like shooting are taken from the previous position of the agent
		// 1. Non-movement actions (shoot, etc.)
		// 2. Movement actions

		// 1. No movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "shoot":
				{
					var vec []float64
					err := json.Unmarshal(mutation.GetArguments(), &vec)
					if err != nil {
						utils.Debug("arenaserver-mutation", "Failed to unmarshal JSON arguments for shoot mutation, coming from agent "+batch.AgentId.String()+"; "+err.Error())
						continue
					}

					nbmutations++
					newstate = mutationShoot(newstate, serverstate, vector.MakeVector2(vec[0], vec[1]))

					break
				}
			}
		}

		// 2. Movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "steer":
				{
					var vec []float64
					err := json.Unmarshal(mutation.GetArguments(), &vec)
					if err != nil {
						utils.Debug("arenaserver-mutation", "Failed to unmarshal JSON arguments for steer mutation, coming from agent "+batch.AgentId.String()+"; "+err.Error())
						continue
					}

					nbmutations++
					newstate = mutationSteer(newstate, vector.MakeVector2(vec[0], vec[1]))

					break
				}
			}
		}

		serverstate.Agentsmutex.Lock()
		serverstate.Agents[batch.AgentId] = newstate
		serverstate.Agentsmutex.Unlock()

	}
}

func initializeMapMemoization(arenaMap *mapcontainer.MapContainer) *MapMemoization {

	///////////////////////////////////////////////////////////////////////////
	// Obstacles
	///////////////////////////////////////////////////////////////////////////

	obstacles := make([]entities.Obstacle, 0)

	// Obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		for _, polygon := range ground.Outline {
			for i := 0; i < len(polygon.Points)-1; i++ {
				a := polygon.Points[i]
				b := polygon.Points[i+1]

				obstacles = append(obstacles, entities.MakeObstacle(
					ground.Id,
					entities.ObstacleType.Ground,
					vector.MakeVector2(a.X, a.Y),
					vector.MakeVector2(b.X, b.Y),
				))
			}
		}
	}

	// Explicit obstacles
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		for i := 0; i < len(polygon.Points)-1; i++ {
			a := polygon.Points[i]
			b := polygon.Points[i+1]
			obstacles = append(obstacles, entities.MakeObstacle(
				obstacle.Id,
				entities.ObstacleType.Object,
				vector.MakeVector2(a.X, a.Y),
				vector.MakeVector2(b.X, b.Y),
			))
		}
	}

	return &MapMemoization{
		Obstacles: obstacles,
	}
}

func buildPhysicalWorld(arenaMap *mapcontainer.MapContainer) *box2d.B2World {

	// Define the gravity vector.
	gravity := box2d.MakeB2Vec2(0.0, 0.0) // 0: the simulation is seen from the top

	// Construct a world object, which will hold and simulate the rigid bodies.
	world := box2d.MakeB2World(gravity)

	// Static obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		for _, polygon := range ground.Outline {

			bodydef := box2d.MakeB2BodyDef()
			bodydef.Type = box2d.B2BodyType.B2_staticBody

			body := world.CreateBody(&bodydef)
			vertices := make([]box2d.B2Vec2, len(polygon.Points)-1) // -1: avoid last point because the last point of the loop should not be repeated

			for i := 0; i < len(polygon.Points)-1; i++ {
				vertices[i].Set(polygon.Points[i].X, polygon.Points[i].Y)
			}

			shape := box2d.MakeB2ChainShape()
			shape.CreateLoop(vertices, len(vertices))
			body.CreateFixture(&shape, 0.0)
			body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Ground, ground.Id))
		}
	}

	// Explicit obstacles
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		bodydef := box2d.MakeB2BodyDef()
		bodydef.Type = box2d.B2BodyType.B2_staticBody

		body := world.CreateBody(&bodydef)
		vertices := make([]box2d.B2Vec2, len(polygon.Points)-1) // a polygon has as many edges as points

		for i := 0; i < len(polygon.Points)-1; i++ {
			vertices[i].Set(polygon.Points[i].X, polygon.Points[i].Y)
		}

		shape := box2d.MakeB2ChainShape()
		shape.CreateLoop(vertices, len(vertices))
		body.CreateFixture(&shape, 0.0)
		body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Obstacle, obstacle.Id))
	}

	return &world
}

func mutationSteer(agentstate entities.AgentState, steering vector.Vector2) entities.AgentState {

	prevmag := agentstate.GetVelocity().Mag()
	diff := steering.Mag() - prevmag
	if math.Abs(diff) > agentstate.MaxSteeringForce {
		if diff > 0 {
			steering = steering.SetMag(prevmag + agentstate.MaxSteeringForce)
		} else {
			steering = steering.SetMag(prevmag - agentstate.MaxSteeringForce)
		}
	}
	abssteering := trigo.LocalAngleToAbsoluteAngleVec(agentstate.GetOrientation(), steering, &agentstate.MaxAngularVelocity)
	agentstate.SetVelocity(abssteering.Limit(agentstate.MaxSpeed))

	return agentstate
}

func mutationShoot(agentstate entities.AgentState, serverstate *ServerState, aiming vector.Vector2) entities.AgentState {

	//
	// Levels consumption
	//

	if agentstate.LastShot <= agentstate.ShootCooldown {
		// invalid shot, cooldown not over
		return agentstate
	}

	if agentstate.ShootEnergy < agentstate.ShootEnergyCost {
		// TODO(jerome): puiser dans le shield ?
		return agentstate
	}

	agentstate.LastShot = 0
	agentstate.ShootEnergy -= agentstate.ShootEnergyCost

	projectileId := uuid.NewV4()

	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////
	// Make physical body for projectile
	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////

	agentpos := agentstate.GetPosition()

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	bodydef.Position.Set(agentpos.GetX(), agentpos.GetY())

	// // on passe le vecteur de visée d'un angle relatif à un angle absolu
	absaiming := trigo.LocalAngleToAbsoluteAngleVec(agentstate.GetOrientation(), aiming, nil) // TODO: replace nil here by an actual angle constraint

	// FIXME(jerome): handle proper Box2D <=> BA velocity conversion
	pvel := absaiming.SetMag(100) // projectile speed; 60 is 3u/tick
	bodydef.LinearVelocity = box2d.MakeB2Vec2(pvel.GetX(), pvel.GetY())

	body := serverstate.PhysicalWorld.CreateBody(&bodydef)
	body.SetLinearDamping(0.0) // no aerodynamic drag

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(0.3)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Projectile, projectileId.String()))
	body.SetBullet(true)

	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////

	projectile := entities.NewBallisticProjectile(projectileId, body)
	projectile.AgentEmitterId = agentstate.GetAgentId()
	projectile.JustFired = true
	projectile.TTL = 60

	serverstate.SetProjectile(projectile.Id, projectile)

	return agentstate
}
