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

func onArenaHandshake(state *State, payload *types.MQPayload) {
	id, ok := (*payload)["id"].(string)

	if ok {
		state.idleArenas[id] = ArenaState{
			id: id,
		}

		utils.Debug("master", id+" joined "+getMasterStatus(state))
	}
}

func onArenaLaunch(state *State, payload *types.MQPayload, client *mq.Client) {

	if len(state.idleArenas) > 0 {

		var arena ArenaState

		for _, value := range state.idleArenas {
			arena = value
			break
		}

		delete(state.idleArenas, arena.id)
		state.runningArenas[arena.id] = arena

		if id, ok := (*payload)["id"].(string); ok {

			client.Publish("arena", arena.id+".launch", types.MQPayload{
				"id": id,
			})

			utils.Debug("master", "Launched arena "+arena.id+" "+getMasterStatus(state))
		}
	} else {
		utils.Debug("master", "No arena available")
	}
}

func onArenaStop(state *State, payload *types.MQPayload) {

	if id, ok := (*payload)["id"].(string); ok {
		arena, ok := state.runningArenas[id]

		if ok {
			delete(state.runningArenas, arena.id)

			utils.Debug("master", "Arena "+id+" stoped "+getMasterStatus(state))
		} else {
			utils.Debug("master", "Arena "+id+" is not running")
		}
	}
}
