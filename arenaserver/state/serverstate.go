package state

import (
	"encoding/json"
	"strconv"
	"sync"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/projectile"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type ServerState struct {
	Agents      map[uuid.UUID](AgentState)
	Agentsmutex *sync.Mutex

	Projectiles                map[uuid.UUID](*projectile.BallisticProjectile)
	Projectilesmutex           *sync.Mutex
	ProjectilesDeletedThisTick map[uuid.UUID](*projectile.BallisticProjectile)

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
		Agents:      make(map[uuid.UUID](AgentState)),
		Agentsmutex: &sync.Mutex{},

		Projectiles:                make(map[uuid.UUID]*projectile.BallisticProjectile),
		Projectilesmutex:           &sync.Mutex{},
		ProjectilesDeletedThisTick: make(map[uuid.UUID]*projectile.BallisticProjectile),

		pendingmutations: make([]protocol.AgentMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},

		DebugPoints:      make([]vector.Vector2, 0),
		debugPointsMutex: &sync.Mutex{},
		MapMemoization:   initializeMapMemoization(arenaMap),
		PhysicalWorld:    buildPhysicalWorld(arenaMap),
	}
}

func (serverstate *ServerState) GetProjectile(projectileid uuid.UUID) *projectile.BallisticProjectile {
	serverstate.Projectilesmutex.Lock()
	res := serverstate.Projectiles[projectileid]
	serverstate.Projectilesmutex.Unlock()

	return res
}

func (serverstate *ServerState) SetProjectile(projectileid uuid.UUID, projectile *projectile.BallisticProjectile) {
	serverstate.Projectilesmutex.Lock()
	serverstate.Projectiles[projectileid] = projectile
	serverstate.Projectilesmutex.Unlock()
}

func (serverstate *ServerState) GetAgentState(agentid uuid.UUID) AgentState {
	serverstate.Agentsmutex.Lock()
	res := serverstate.Agents[agentid]
	serverstate.Agentsmutex.Unlock()

	return res
}

func (serverstate *ServerState) SetAgentState(agentid uuid.UUID, agentstate AgentState) {
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
		newstate := agentstate.clone()
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
					newstate = newstate.mutationShoot(serverstate, vector.MakeVector2(vec[0], vec[1]))

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
					newstate = newstate.mutationSteer(vector.MakeVector2(vec[0], vec[1]))

					break
				}
			}
		}

		if newstate.validate() && newstate.validateTransition(agentstate) {
			serverstate.Agentsmutex.Lock()
			serverstate.Agents[batch.AgentId] = newstate
			serverstate.Agentsmutex.Unlock()
		} else {
			utils.Debug("core-loop", "ILLEGAL Mutations "+strconv.Itoa(nbmutations))
		}
	}
}

func initializeMapMemoization(arenaMap *mapcontainer.MapContainer) *MapMemoization {

	///////////////////////////////////////////////////////////////////////////
	// Obstacles
	///////////////////////////////////////////////////////////////////////////

	obstacles := make([]Obstacle, 0)

	// Obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		for _, polygon := range ground.Outline {
			for i := 0; i < len(polygon.Points)-1; i++ {
				a := polygon.Points[i]
				b := polygon.Points[i+1]

				obstacles = append(obstacles, MakeObstacle(
					ground.Id,
					ObstacleType.Ground,
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
			obstacles = append(obstacles, MakeObstacle(
				obstacle.Id,
				ObstacleType.Object,
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
