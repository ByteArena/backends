package arenamaster

import (
	"log"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
)

type onLogicResponseCallable func(*mq.Client)
type onLogic func(state *State, payload *types.MQPayload) onLogicResponseCallable

func onArenaHandshake(state *State, payload *types.MQPayload) onLogicResponseCallable {
	uuid, ok := (*payload)["uuid"].(string)

	if ok {
		state.arenas = append(state.arenas, ArenaState{
			uuid: uuid,
		})

		log.Println(uuid + " joined the pool")
	}

	return nil
}

func onArenaLaunch(state *State, payload *types.MQPayload) onLogicResponseCallable {
	log.Println(state.arenas)
	if len(state.arenas) > 0 {
		arena := state.arenas[0]

		state.arenas = state.arenas[1:]

		return func(client *mq.Client) {
			var msg struct{}
			client.Publish("arena", arena.uuid+".launch", msg)
		}
	}

	log.Println("No arena available")

	return nil
}
