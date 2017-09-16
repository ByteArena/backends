package arenamaster

import (
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func onGameHandshake(state *State, payload *types.MQPayload) {
	id, ok := (*payload)["id"].(string)

	if ok {
		state.LockState()
		defer state.UnlockState()

		state.idleArenas[id] = ArenaServerState{
			id: id,
		}

		utils.Debug("master", id+" joined "+getMasterStatus(state))
	}
}
