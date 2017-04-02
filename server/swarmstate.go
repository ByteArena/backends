package server

import (
	"log"
	"strconv"

	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
	"github.com/scryner/lfreequeue"
)

type SwarmState struct {
	pin              *utils.Vector2
	Agents           map[uuid.UUID](*AgentState)
	pendingmutations *lfreequeue.Queue
}

/* ***************************************************************************/
/* SwarmState implementation */
/* ***************************************************************************/

func NewSwarmState() *SwarmState {
	return &SwarmState{
		Agents:           make(map[uuid.UUID](*AgentState)),
		pin:              utils.NewVector2(200, 300).Clone(),
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

		log.Println("Processing mutations on turn " + strconv.Itoa(int(batch.Turn)) + " for agent " + batch.Agent.id.String())

		for _, mutation := range batch.Mutations {
			switch mutation.action {
			/*case "mutationIncrement":
			{
				nbmutations++
				newstate.mutationIncrement()
				break
			}
			*/
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
					//newstate.mutationAccelerate(RandomVector2())
					break
				}
			}
		}

		//statejson, _ := json.Marshal(newstate)

		if newstate.validate() && newstate.validateTransition(agentstate) {
			swarmstate.Agents[batch.Agent.id] = newstate
			//log.Println("Mutations LEGALES " + strconv.Itoa(nbmutations) + "; state: " + string(statejson))
		} else {
			//log.Println("Mutations ILLEGALES " + strconv.Itoa(nbmutations) + "; state: " + string(statejson))
		}
	}

	/*if nbmutations != 8 {
		log.Println("ERREUR --------------------- " + strconv.Itoa(nbmutations) + ", expected 8")
	}*/
}
