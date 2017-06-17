package arenamaster

import (
	"log"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
)

type onLogicResponseCallable func(*mq.Client)
type onLogic func(state *State, payload *types.MQPayload) onLogicResponseCallable

func onArenaHandshake(state *State, payload *types.MQPayload) onLogicResponseCallable {
	id, ok := (*payload)["id"].(string)

	if ok {
		state.arenas = append(state.arenas, ArenaState{
			id: id,
		})

		log.Println(id + " joined the pool")
	}

	return nil
}

func onArenaLaunch(state *State, payload *types.MQPayload) onLogicResponseCallable {
	log.Println(state.arenas)
	if len(state.arenas) > 0 {
		arena := state.arenas[0]

		state.arenas = state.arenas[1:]

		return func(client *mq.Client) {

			if id, ok := (*payload)["id"].(string); ok {
				client.Publish("arena", arena.id+".launch", types.MQPayload{
					"id": id,
				})
			}
		}
	}

	log.Println("No arena available")

	return nil
}
