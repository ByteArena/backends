package server

import (
	"log"
	"math/rand"
	"strconv"

	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
	"github.com/scryner/lfreequeue"
)

type SwarmState struct {
	Pin              *utils.Vector2
	PinCenter        *utils.Vector2
	Agents           map[uuid.UUID](*AgentState)
	pendingmutations *lfreequeue.Queue
}

/* ***************************************************************************/
/* SwarmState implementation */
/* ***************************************************************************/

func NewSwarmState() *SwarmState {

	pin := utils.NewVector2(rand.Float64()*300+100, rand.Float64()*300+100)

	return &SwarmState{
		Agents:           make(map[uuid.UUID](*AgentState)),
		Pin:              pin.Clone(),
		PinCenter:        pin.Clone(),
		pendingmutations: lfreequeue.NewQueue(),
	}
}

func (swarmstate *SwarmState) PushMutationBatch(batch *StateMutationBatch) {
	swarmstate.pendingmutations.Enqueue(batch)
}

func (swarmstate *SwarmState) ProcessMutation() {
	for _batch := range swarmstate.pendingmutations.Iter() {
		batch, ok := _batch.(*StateMutationBatch)
		if !ok {
			continue
		}

		nbmutations := 0

		agentstate := swarmstate.Agents[batch.Agent.id]
		newstate := agentstate.clone()

		log.Println("Processing mutations on " + batch.Turn.String() + " for agent " + batch.Agent.String())

		for _, mutation := range batch.Mutations {
			switch mutation.action {
			case "mutationSteer":
				{
					log.Println(mutation.arguments[0])
					vec, ok := mutation.arguments[0].([]interface{})
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
					newstate.mutationSteer(utils.NewVector2(x, y))
					break
				}
			}
		}

		if newstate.validate() && newstate.validateTransition(agentstate) {
			swarmstate.Agents[batch.Agent.id] = newstate
		} else {
			log.Println("Mutations ILLEGALES " + strconv.Itoa(nbmutations) + ";")
		}
	}
}
