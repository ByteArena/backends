package state

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"

	"github.com/bytearena/bytearena/arenaserver/projectile"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
	uuid "github.com/satori/go.uuid"
)

type ServerState struct {
	Agents      map[uuid.UUID](AgentState)
	Agentsmutex *sync.Mutex

	Projectiles      map[uuid.UUID](*projectile.BallisticProjectile)
	Projectilesmutex *sync.Mutex

	pendingmutations []protocol.StateMutationBatch
	mutationsmutex   *sync.Mutex

	DebugPoints      []vector.Vector2
	debugPointsMutex *sync.Mutex

	MapMemoization *MapMemoization
}

/* ***************************************************************************/
/* ServerState implementation */
/* ***************************************************************************/

func NewServerState(arenaMap *mapcontainer.MapContainer) *ServerState {

	return &ServerState{
		Agents:      make(map[uuid.UUID](AgentState)),
		Agentsmutex: &sync.Mutex{},

		Projectiles:      make(map[uuid.UUID]*projectile.BallisticProjectile),
		Projectilesmutex: &sync.Mutex{},

		pendingmutations: make([]protocol.StateMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},

		DebugPoints:      make([]vector.Vector2, 0),
		debugPointsMutex: &sync.Mutex{},

		MapMemoization: InitializeMapMemoization(arenaMap),
	}
}

func InitializeMapMemoization(arenaMap *mapcontainer.MapContainer) *MapMemoization {
	// We have to initialize the Obstacle list
	obstacles := make([]Obstacle, 0)

	// Les sols
	for _, ground := range arenaMap.Data.Grounds {
		for _, polygon := range ground.Outline {
			for i := 0; i < len(polygon.Points)-1; i++ {
				a := polygon.Points[i]
				b := polygon.Points[i+1]
				obstacles = append(obstacles, MakeObstacle(
					vector.MakeVector2(a.X, a.Y),
					vector.MakeVector2(b.X, b.Y),
				))
			}
		}
	}

	// Les obstacles explicites
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		for i := 0; i < len(polygon.Points)-1; i++ {
			a := polygon.Points[i]
			b := polygon.Points[i+1]
			obstacles = append(obstacles, MakeObstacle(
				vector.MakeVector2(a.X, a.Y),
				vector.MakeVector2(b.X, b.Y),
			))
		}
	}

	///////////////////////////////////////////////////////////////////////////
	// Initialize RTree
	///////////////////////////////////////////////////////////////////////////

	rt := rtreego.NewTree(2, 25, 50) // TODO: better constants here ? what heuristic to use ?

	for _, obstacle := range obstacles {

		pa, pb := GetBoundingBox([]vector.Vector2{obstacle.A, obstacle.B})
		r, err := rtreego.NewRect(pa, pb)
		utils.CheckWithFunc(err, func() string {
			return "rtreego: NewRect error;" + err.Error()
		})

		rt.Insert(&GeometryObject{
			Type:   GeometryObjectType.Obstacle,
			ID:     obstacle.Id.String(),
			Rect:   r,
			PointA: obstacle.A,
			PointB: obstacle.B,
		})
	}

	return &MapMemoization{
		Obstacles:      obstacles,
		RtreeObstacles: rt,
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
					utils.Check(err, "Failed to unmarshal JSON arguments for steer mutation, coming from agent "+batch.AgentId.String())

					nbmutations++
					newstate = newstate.mutationSteer(vector.MakeVector2(vec[0], vec[1]))

					break
				}
			case "shoot":
				{
					var vec []float64
					err := json.Unmarshal(mutation.GetArguments(), &vec)
					utils.Check(err, "Failed to unmarshal JSON arguments for shoot mutation, coming from agent "+batch.AgentId.String())

					nbmutations++
					newstate = newstate.mutationShoot(serverstate, vector.MakeVector2(vec[0], vec[1]))

					break
				}
			case "debugvis":
				{
					var rawvecs [][]float64
					err := json.Unmarshal(mutation.GetArguments(), &rawvecs)
					utils.Check(err, "Failed to unmarshal JSON arguments for debugvis mutation, coming from agent "+batch.AgentId.String())

					if len(rawvecs) > 0 {
						vecs := make([]vector.Vector2, len(rawvecs))
						for i, rawvec := range rawvecs {
							v := vector.MakeVector2(rawvec[0], rawvec[1])
							vecs[i] = v.SetAngle(v.Angle() + agentstate.Orientation).Add(agentstate.Position)
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
			log.Println("Mutations ILLEGALES " + strconv.Itoa(nbmutations) + ";")
		}
	}
}

var GeometryObjectType = struct {
	Obstacle   uint8
	Agent      uint8
	Projectile uint8
}{
	Obstacle:   0,
	Agent:      1,
	Projectile: 2,
}

type GeometryObject struct {
	ID     string
	Type   uint8
	Rect   *rtreego.Rect
	PointA vector.Vector2
	PointB vector.Vector2
}

func (geobj GeometryObject) Bounds() *rtreego.Rect {
	return geobj.Rect
}

func NewGeometryObjectID(id string) *GeometryObject {
	return &GeometryObject{
		ID: id,
	}
}

func GetBoundingBox(points []vector.Vector2) (rtreego.Point, rtreego.Point) {

	var minX, minY *float64
	var maxX, maxY *float64

	for _, point := range points {
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

	return []float64{*minX, *minY}, []float64{*maxX - *minX + 0.00001, *maxY - *minY + 0.00001}
}
