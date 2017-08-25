package arenamaster

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func onGameLaunched(state *State, payload *types.MQPayload, mqclient *mq.Client, gql *graphql.Client) {
	if arenaServerUUID, ok := (*payload)["arenaserveruuid"].(string); ok {
		if arena, ok := state.pendingArenas[arenaServerUUID]; ok {

			// Put it into running arenas, now that we're sure
			state.runningArenas[arena.id] = arena

			delete(state.pendingArenas, arena.id)

			utils.Debug("master", arenaServerUUID+" launched "+getMasterStatus(state))
		}
	}
}
