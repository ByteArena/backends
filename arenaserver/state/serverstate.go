package state

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"

	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type ServerState struct {
	Agents      map[uuid.UUID](AgentState)
	Agentsmutex *sync.Mutex

	Projectiles      map[uuid.UUID](ProjectileState)
	Projectilesmutex *sync.Mutex

	//Obstacles      []Obstacle
	//Obstaclesmutex *sync.Mutex

	pendingmutations []protocol.StateMutationBatch
	mutationsmutex   *sync.Mutex

	DebugIntersects         []vector.Vector2
	DebugIntersectsRejected []vector.Vector2

	DebugPoints      []vector.Vector2
	debugPointsMutex *sync.Mutex

	MapMemoization *MapMemoization
}

/* ***************************************************************************/
/* ServerState implementation */
/* ***************************************************************************/

func NewServerState() *ServerState {

	return &ServerState{
		Agents:      make(map[uuid.UUID](AgentState)),
		Agentsmutex: &sync.Mutex{},

		Projectiles:      make(map[uuid.UUID](ProjectileState)),
		Projectilesmutex: &sync.Mutex{},

		//Obstacles:      make([]Obstacle, 0),
		//Obstaclesmutex: &sync.Mutex{},

		pendingmutations: make([]protocol.StateMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},

		DebugIntersects:         make([]vector.Vector2, 0),
		DebugIntersectsRejected: make([]vector.Vector2, 0),

		DebugPoints:      make([]vector.Vector2, 0),
		debugPointsMutex: &sync.Mutex{},

		//DebugSegments: []vector.Vector2,
		//debugSegmentsMutex: &sync.Mutex{},
	}
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

// func (serverstate *ServerState) SetObstacle(obstacle Obstacle) {
// 	serverstate.Obstaclesmutex.Lock()
// 	serverstate.Obstacles = append(serverstate.Obstacles, obstacle)
// 	serverstate.Obstaclesmutex.Unlock()
// }

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

		//log.Println("Processing mutations on " + batch.Turn.String() + " for agent " + batch.AgentId.String())

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
