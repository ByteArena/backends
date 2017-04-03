package server

import (
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type SwarmState struct {
	Pin              utils.Vector2
	PinCenter        utils.Vector2
	Agents           map[uuid.UUID](AgentState)
	mutationsmutex   *sync.Mutex
	pendingmutations []StateMutationBatch
}

/* ***************************************************************************/
/* SwarmState implementation */
/* ***************************************************************************/

func NewSwarmState() *SwarmState {

	pin := utils.MakeVector2(rand.Float64()*300+100, rand.Float64()*300+100)

	return &SwarmState{
		Agents:           make(map[uuid.UUID](AgentState)),
		Pin:              pin,
		PinCenter:        pin,
		mutationsmutex:   &sync.Mutex{},
		pendingmutations: make([]StateMutationBatch, 0),
	}
}

func (swarmstate *SwarmState) PushMutationBatch(batch StateMutationBatch) {
	swarmstate.mutationsmutex.Lock()
	swarmstate.pendingmutations = append(swarmstate.pendingmutations, batch)
	swarmstate.mutationsmutex.Unlock()
}

func (swarmstate *SwarmState) ProcessMutations() {

	swarmstate.mutationsmutex.Lock()
	mutations := swarmstate.pendingmutations
	swarmstate.pendingmutations = make([]StateMutationBatch, 0)
	swarmstate.mutationsmutex.Unlock()

	for _, batch := range mutations {

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
					newstate = newstate.mutationSteer(utils.MakeVector2(x, y))
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
