package state

import (
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/netgusto/bytearena/server/statemutation"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type ServerState struct {
	Pin              utils.Vector2
	PinCenter        utils.Vector2
	Agents           map[uuid.UUID](AgentState)
	Projectiles      map[uuid.UUID](ProjectileState)
	mutationsmutex   *sync.Mutex
	pendingmutations []statemutation.StateMutationBatch
}

/* ***************************************************************************/
/* SwarmState implementation */
/* ***************************************************************************/

func NewServerState() *ServerState {

	pin := utils.MakeVector2(rand.Float64()*300+100, rand.Float64()*300+100)

	return &ServerState{
		Agents:           make(map[uuid.UUID](AgentState)),
		Projectiles:      make(map[uuid.UUID](ProjectileState)),
		Pin:              pin,
		PinCenter:        pin,
		mutationsmutex:   &sync.Mutex{},
		pendingmutations: make([]statemutation.StateMutationBatch, 0),
	}
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

		log.Println("Processing mutations on " + batch.Turn.String() + " for agent " + batch.AgentId.String())

		for _, mutation := range batch.Mutations {
			switch mutation.Action {
			case "steer":
				{
					vec, ok := mutation.Arguments[0].([]interface{})
					if !ok {
						log.Panicln("Invalid mutationSteer argument")
					}

					x, ok := vec[0].(float64)
					if !ok {
						log.Panicln("Invalid mutationSteer argument")
					}

					y, ok := vec[1].(float64)
					if !ok {
						log.Panicln("Invalid mutationSteer argument")
					}

					nbmutations++
					newstate = newstate.mutationSteer(utils.MakeVector2(x, y))

					break
				}

			case "shoot":
				{
					vec, ok := mutation.Arguments[0].([]interface{})
					if !ok {
						log.Panicln("Invalid mutationShoot argument")
					}

					x, ok := vec[0].(float64)
					if !ok {
						log.Panicln("Invalid mutationShoot argument")
					}

					y, ok := vec[1].(float64)
					if !ok {
						log.Panicln("Invalid mutationShoot argument")
					}

					nbmutations++

					agentX, agentY := newstate.Position.Get()

					projectile := ProjectileState{
						Position: utils.MakeVector2(agentX+newstate.Radius, agentY+newstate.Radius),
						Velocity: newstate.Position.Add(utils.MakeVector2(x, y)), // adding the agent position to "absolutize" the target vector
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
