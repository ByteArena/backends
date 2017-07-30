package arenamaster

import (
	"log"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
)

func onArenaHandshake(state *State, payload *types.MQPayload) {
	id, ok := (*payload)["id"].(string)

	if ok {
		state.arenas = append(state.arenas, ArenaState{
			id: id,
		})

		log.Println(id + " joined the pool")
	}
}

func onArenaLaunch(state *State, payload *types.MQPayload, client *mq.Client) {
	log.Println(state.arenas)

	if len(state.arenas) > 0 {
		arena := state.arenas[0]
		state.arenas = state.arenas[1:]

		if id, ok := (*payload)["id"].(string); ok {

			client.Publish("arena", arena.id+".launch", types.MQPayload{
				"id": id,
			})
		}
	}

	log.Println("No arena available")
}
