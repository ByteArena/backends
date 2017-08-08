package arenamaster

import (
	"strconv"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func getMasterStatus(state *State) string {
	return "(" + strconv.Itoa(len(state.idleArenas)) + " arena(s) idle, " + strconv.Itoa(len(state.runningArenas)) + " arena(s) running)"
}

func onGameHandshake(state *State, payload *types.MQPayload) {
	id, ok := (*payload)["id"].(string)

	if ok {
		state.idleArenas[id] = ArenaState{
			id: id,
		}

		utils.Debug("master", id+" joined "+getMasterStatus(state))
	}
}

func onGameLaunch(state *State, payload *types.MQPayload, client *mq.Client) {

	if len(state.idleArenas) > 0 {

		var astate ArenaState

		for _, value := range state.idleArenas {
			astate = value
			break
		}

		delete(state.idleArenas, astate.id)
		state.runningArenas[astate.id] = astate

		if id, ok := (*payload)["id"].(string); ok {

			client.Publish("game", astate.id+".launch", types.MQPayload{
				"id": id,
			})

			utils.Debug("master", "Launched game "+astate.id+" "+getMasterStatus(state))
		}
	} else {
		utils.Debug("master", "No game available")
	}
}

func onGameStop(state *State, payload *types.MQPayload) {

	if id, ok := (*payload)["id"].(string); ok {
		arena, ok := state.runningArenas[id]

		if ok {
			delete(state.runningArenas, arena.id)

			utils.Debug("master", "Arena "+id+" stopped "+getMasterStatus(state))
		} else {
			utils.Debug("master", "Arena "+id+" is not running")
		}
	}
}
