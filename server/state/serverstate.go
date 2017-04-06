package state

import (
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/netgusto/bytearena/server/statemutation"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type ServerState struct {
	Pin       utils.Vector2
	PinCenter utils.Vector2

	Agents      map[uuid.UUID](AgentState)
	Agentsmutex *sync.Mutex

	Projectiles      map[uuid.UUID](ProjectileState)
	Projectilesmutex *sync.Mutex

	pendingmutations []statemutation.StateMutationBatch
	mutationsmutex   *sync.Mutex
}

/* ***************************************************************************/
/* ServerState implementation */
/* ***************************************************************************/

func NewServerState() *ServerState {

	pin := utils.MakeVector2(rand.Float64()*300+100, rand.Float64()*300+100)

	return &ServerState{
		Pin:       pin,
		PinCenter: pin,

		Agents:      make(map[uuid.UUID](AgentState)),
		Agentsmutex: &sync.Mutex{},

		Projectiles:      make(map[uuid.UUID](ProjectileState)),
		Projectilesmutex: &sync.Mutex{},

		pendingmutations: make([]statemutation.StateMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},
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

func (serverstate *ServerState) PushMutationBatch(batch statemutation.StateMutationBatch) {
	serverstate.mutationsmutex.Lock()
	serverstate.pendingmutations = append(serverstate.pendingmutations, batch)
	serverstate.mutationsmutex.Unlock()
}

func (serverstate *ServerState) ProcessMutations() {

	serverstate.mutationsmutex.Lock()
	mutations := serverstate.pendingmutations
	serverstate.pendingmutations = make([]statemutation.StateMutationBatch, 0)
	serverstate.mutationsmutex.Unlock()

	for _, batch := range mutations {

		nbmutations := 0

		agentstate := serverstate.Agents[batch.AgentId]
		newstate := agentstate.clone()

		//log.Println("Processing mutations on " + batch.Turn.String() + " for agent " + batch.AgentId.String())

		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "steer":
				{
					var vec []float64
					if err := json.Unmarshal(mutation.GetArguments(), &vec); err != nil {
						log.Panicln(err)
					}

					nbmutations++
					newstate = newstate.mutationSteer(utils.MakeVector2(vec[0], vec[1]))

					break
				}
			case "shoot":
				{
					var vec []float64
					if err := json.Unmarshal(mutation.GetArguments(), &vec); err != nil {
						log.Panicln(err)
					}

					nbmutations++

					agentX, agentY := newstate.Position.Get()

					projectile := ProjectileState{
						Position: utils.MakeVector2(agentX+newstate.Radius, agentY+newstate.Radius),
						Velocity: newstate.Position.Add(utils.MakeVector2(vec[0], vec[1])), // adding the agent position to "absolutize" the target vector
						From:     newstate,
						Ttl:      1,
					}

					projectileid := uuid.NewV4()

					serverstate.Projectilesmutex.Lock()
					serverstate.Projectiles[projectileid] = projectile
					serverstate.Projectilesmutex.Unlock()

					break
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
