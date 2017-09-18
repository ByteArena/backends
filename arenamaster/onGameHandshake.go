package arenamaster

import (
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"log"
)

func onGameHandshake(state *State, payload *types.MQPayload) {
	id, ok := (*payload)["arenaserveruuid"].(string)

	if ok {
		state.LockState()
		defer state.UnlockState()

		state.idleArenas[id] = ArenaServerState{
			id: id,
		}

		utils.Debug("master", id+" joined "+getMasterStatus(state))
	} else {
		log.Println(*payload)
		utils.Debug("master", "Received handshake event but payload is not parsable")
	}
}
