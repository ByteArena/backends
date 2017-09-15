package state

import (
	"encoding/json"
	"strconv"
	"sync"

	b2collision "github.com/bytearena/box2d/box2d/collision"
	b2common "github.com/bytearena/box2d/box2d/common"
	b2dynamics "github.com/bytearena/box2d/box2d/dynamics"
	"github.com/bytearena/bytearena/arenaserver/projectile"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
	uuid "github.com/satori/go.uuid"
)

type ServerState struct {
	Agents      map[uuid.UUID](AgentState)
	Agentsmutex *sync.Mutex

	Projectiles                map[uuid.UUID](*projectile.BallisticProjectile)
	Projectilesmutex           *sync.Mutex
	ProjectilesDeletedThisTick map[uuid.UUID](*projectile.BallisticProjectile)

	pendingmutations []protocol.StateMutationBatch
	mutationsmutex   *sync.Mutex

	DebugPoints      []vector.Vector2
	debugPointsMutex *sync.Mutex

	PhysicalWorld *b2dynamics.B2World

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

		pendingmutations: make([]protocol.StateMutationBatch, 0),
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

func (serverstate *ServerState) PushMutationBatch(batch protocol.StateMutationBatch) {
	serverstate.mutationsmutex.Lock()
	serverstate.pendingmutations = append(serverstate.pendingmutations, batch)
	serverstate.mutationsmutex.Unlock()
}

func (serverstate *ServerState) ProcessMutations() {

	serverstate.mutationsmutex.Lock()
	mutations := serverstate.pendingmutations
	serverstate.pendingmutations = make([]protocol.StateMutationBatch, 0)
	serverstate.mutationsmutex.Unlock()

	for _, batch := range mutations {

		nbmutations := 0

		serverstate.Agentsmutex.Lock()
		agentstate := serverstate.Agents[batch.AgentId]
		newstate := agentstate.clone()
		serverstate.Agentsmutex.Unlock()

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
			case "debugvis":
				{
					var rawvecs [][]float64
					err := json.Unmarshal(mutation.GetArguments(), &rawvecs)
					if err != nil {
						utils.Debug("arenaserver-mutation", "Failed to unmarshal JSON arguments for debugvis mutation, coming from agent "+batch.AgentId.String()+"; "+err.Error())
						continue
					}

					if len(rawvecs) == 2 {
						vecs := make([]vector.Vector2, len(rawvecs))
						for i, rawvec := range rawvecs {
							v := vector.MakeVector2(rawvec[0], rawvec[1])
							vecs[i] = v.SetAngle(v.Angle() + agentstate.GetOrientation()).Add(agentstate.GetPosition())
						}

						serverstate.debugPointsMutex.Lock()
						for _, vec := range vecs {
							serverstate.DebugPoints = append(serverstate.DebugPoints, vec)
						}
						serverstate.debugPointsMutex.Unlock()
					}
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

var GeometryObjectType = struct {
	ObstacleGround int
	ObstacleObject int
	Agent          int
	Projectile     int
}{
	ObstacleGround: 0,
	ObstacleObject: 1,
	Agent:          2,
	Projectile:     3,
}

type GeometryObject struct {
	ID     string
	Type   int
	Rect   *rtreego.Rect
	PointA vector.Vector2
	PointB vector.Vector2
	Normal vector.Vector2
}

func (geobj GeometryObject) Bounds() *rtreego.Rect {
	return geobj.Rect
}

func (geobj *GeometryObject) GetPointA() vector.Vector2 {
	return geobj.PointA
}

func (geobj *GeometryObject) GetPointB() vector.Vector2 {
	return geobj.PointB
}

func (geobj *GeometryObject) GetRadius() float64 {
	return 0
}

func (geobj *GeometryObject) GetType() int {
	return geobj.Type
}

func (geobj *GeometryObject) GetID() string {
	return geobj.ID
}

type TriangleRtreeWrapper struct {
	Rect   *rtreego.Rect
	Points [3]vector.Vector2
}

func (geobj TriangleRtreeWrapper) Bounds() *rtreego.Rect {
	return geobj.Rect
}

func GetBoundingBox(points []vector.Vector2) (rtreego.Point, rtreego.Point) {

	var minX = 10000000000.0
	var minY = 10000000000.0
	var maxX = -10000000000.0
	var maxY = -10000000000.0

	for _, point := range points {
		x, y := point.Get()
		if x < minX {
			minX = x
		}

		if y < minY {
			minY = y
		}

		if x > maxX {
			maxX = x
		}

		if y > maxY {
			maxY = y
		}
	}

	width := maxX - minX
	if width <= 0 {
		width = 0.00001
	}

	height := maxY - minY
	if height <= 0 {
		height = 0.00001
	}

	return []float64{minX, minY}, []float64{width, height}
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
				normal := polygon.Normals[i]
				obstacles = append(obstacles, MakeObstacle(
					ground.Id,
					ObstacleType.Ground,
					vector.MakeVector2(a.X, a.Y),
					vector.MakeVector2(b.X, b.Y),
					vector.MakeVector2(normal.X, normal.Y),
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
			normal := polygon.Normals[i]
			obstacles = append(obstacles, MakeObstacle(
				obstacle.Id,
				ObstacleType.Object,
				vector.MakeVector2(a.X, a.Y),
				vector.MakeVector2(b.X, b.Y),
				vector.MakeVector2(normal.X, normal.Y),
			))
		}
	}

	return &MapMemoization{
		Obstacles: obstacles,
	}
}

func buildPhysicalWorld(arenaMap *mapcontainer.MapContainer) *b2dynamics.B2World {

	// Define the gravity vector.
	gravity := b2common.MakeB2Vec2(0.0, 0.0) // 0: the simulation is seen from the top

	// Construct a world object, which will hold and simulate the rigid bodies.
	world := b2dynamics.MakeB2World(gravity)

	// Static obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		for _, polygon := range ground.Outline {

			bodydef := b2dynamics.MakeB2BodyDef()
			bodydef.Type = b2dynamics.B2BodyType.B2_staticBody

			body := world.CreateBody(&bodydef)
			vertices := make([]b2common.B2Vec2, len(polygon.Points)-1) // -1: avoid last point because the last point of the loop should not be repeated

			for i := 0; i < len(polygon.Points)-1; i++ {
				vertices[i].Set(polygon.Points[i].X, polygon.Points[i].Y)
			}

			shape := b2collision.MakeB2ChainShape()
			shape.CreateLoop(vertices, len(vertices))
			body.CreateFixtureFromShapeAndDensity(&shape, 0.0)
			body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Ground, ground.Id))
		}
	}

	// Explicit obstacles
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		bodydef := b2dynamics.MakeB2BodyDef()
		bodydef.Type = b2dynamics.B2BodyType.B2_staticBody

		body := world.CreateBody(&bodydef)
		vertices := make([]b2common.B2Vec2, len(polygon.Points)-1) // a polygon has as many edges as points

		for i := 0; i < len(polygon.Points)-1; i++ {
			vertices[i].Set(polygon.Points[i].X, polygon.Points[i].Y)
		}

		shape := b2collision.MakeB2ChainShape()
		shape.CreateLoop(vertices, len(vertices))
		body.CreateFixtureFromShapeAndDensity(&shape, 0.0)
		body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Obstacle, obstacle.Id))
	}

	return &world
}
