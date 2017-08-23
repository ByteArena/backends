package arenamaster

import (
	"strconv"

	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func getMasterStatus(state *State) string {
	return "(" + strconv.Itoa(len(state.idleArenas)) + " arena(s) idle, " + strconv.Itoa(len(state.runningArenas)) + " arena(s) running, " + strconv.Itoa(len(state.pendingArenas)) + " arena(s) pending)"
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
