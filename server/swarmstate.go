package main

import (
	"encoding/json"
	"log"
	"strconv"

	uuid "github.com/satori/go.uuid"
	"github.com/scryner/lfreequeue"
)

type SwarmState struct {
	agents           map[uuid.UUID](*AgentState)
	pendingmutations *lfreequeue.Queue
}

/* ***************************************************************************/
/* SwarmState implementation */
/* ***************************************************************************/

func NewSwarmState() *SwarmState {
	return &SwarmState{
		agents:           make(map[uuid.UUID](*AgentState)),
		pendingmutations: lfreequeue.NewQueue(),
	}
}

func (swarmstate *SwarmState) PushMutationBatch(batch *MutationBatch) {
	swarmstate.pendingmutations.Enqueue(batch)
}

func (swarmstate *SwarmState) ProcessMutation() {
	for _batch := range swarmstate.pendingmutations.Iter() {
		batch, ok := _batch.(*MutationBatch)
		if !ok {
			continue
		}

		nbmutations := 0

		agentstate := swarmstate.agents[batch.Agent.id]
		newstate := agentstate.clone()

		log.Println("Processing mutations on turn " + strconv.Itoa(int(batch.Turn)) + " for agent " + batch.Agent.id.String())

		for _, mutation := range batch.Mutations {
			switch mutation.action {
			case "mutationIncrement":
				{
					nbmutations++
					newstate.mutationIncrement()
					break
				}
			case "mutationAccelerate":
				{
					/*vec, ok := mutation.arguments[0].([]interface{})
					if !ok {
						log.Panicln("Invalid mutationAccelerate argument")
					}

					x, ok := vec[0].(float64)
					if !ok {
						log.Panicln("Invalid mutationAccelerate argument")
					}

					y, ok := vec[1].(float64)
					if !ok {
						log.Panicln("Invalid mutationAccelerate argument")
					}*/

					nbmutations++
					//newstate.mutationAccelerate(Vector2{x, y})
					newstate.mutationAccelerate(RandomVector2())
					break
				}
			}
		}

		statejson, _ := json.Marshal(newstate)

		if newstate.validate() && newstate.validateTransition(agentstate) {
			swarmstate.agents[batch.Agent.id] = newstate
			log.Println("Mutations LEGALES " + strconv.Itoa(nbmutations) + "; state: " + string(statejson))
		} else {
			log.Println("Mutations ILLEGALES " + strconv.Itoa(nbmutations) + "; state: " + string(statejson))
		}
	}

	/*if nbmutations != 8 {
		log.Println("ERREUR --------------------- " + strconv.Itoa(nbmutations) + ", expected 8")
	}*/
}
