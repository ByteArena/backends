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
	agentsmutex *sync.Mutex

	Projectiles      map[uuid.UUID](ProjectileState)
	projectilesmutex *sync.Mutex

	pendingmutations []statemutation.StateMutationBatch
	mutationsmutex   *sync.Mutex
}

/* ***************************************************************************/
/* SwarmState implementation */
/* ***************************************************************************/

func NewServerState() *ServerState {

	pin := utils.MakeVector2(rand.Float64()*300+100, rand.Float64()*300+100)

	return &ServerState{
		Pin:       pin,
		PinCenter: pin,

		Agents:      make(map[uuid.UUID](AgentState)),
		agentsmutex: &sync.Mutex{},

		Projectiles:      make(map[uuid.UUID](ProjectileState)),
		projectilesmutex: &sync.Mutex{},

		pendingmutations: make([]statemutation.StateMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},
	}
}

func (swarmstate *ServerState) SetAgentState(agentid uuid.UUID, agentstate AgentState) {
	swarmstate.agentsmutex.Lock()
	swarmstate.Agents[agentid] = agentstate
	swarmstate.agentsmutex.Unlock()
}

func (swarmstate *ServerState) PushMutationBatch(batch statemutation.StateMutationBatch) {
	swarmstate.mutationsmutex.Lock()
	swarmstate.pendingmutations = append(swarmstate.pendingmutations, batch)
	swarmstate.mutationsmutex.Unlock()
}

func (swarmstate *ServerState) ProcessMutations() {

	swarmstate.mutationsmutex.Lock()
	mutations := swarmstate.pendingmutations
	swarmstate.pendingmutations = make([]statemutation.StateMutationBatch, 0)
	swarmstate.mutationsmutex.Unlock()

	for _, batch := range mutations {

		nbmutations := 0

		agentstate := swarmstate.Agents[batch.AgentId]
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

					swarmstate.Projectiles[projectileid] = projectile

					break
				}
			}
		}

		if newstate.validate() && newstate.validateTransition(agentstate) {
			swarmstate.Agents[batch.AgentId] = newstate
		} else {
			log.Println("Mutations ILLEGALES " + strconv.Itoa(nbmutations) + ";")
		}
	}
}
