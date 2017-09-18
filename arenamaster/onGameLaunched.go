package arenamaster

import (
	"log"
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func onGameLaunched(state *State, payload *types.MQPayload, mqclient *mq.Client, gql *graphql.Client) {
	log.Println(*payload)

	if arenaServerUUID, ok := (*payload)["arenaserveruuid"].(string); ok {

		state.LockState()
		defer state.UnlockState()

		if arenaServer, ok := state.pendingArenas[arenaServerUUID]; ok {

			// Put it into running arenas, now that we're sure
			state.runningArenas[arenaServer.id] = arenaServer

			// Remove it from pending arenas
			delete(state.pendingArenas, arenaServer.id)

			utils.Debug("master", arenaServerUUID+" launched "+getMasterStatus(state))

			// syncing state in graphql db
			go func() {
				gameid, _ := (*payload)["id"].(string)
				_, err := gql.RequestSync(
					graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
						"id": gameid,
						"game": graphql.Variables{
							"runStatus":       gqltypes.GameRunStatus.Running,
							"launchedAt":      time.Now().Format(time.RFC822Z),
							"arenaServerUUID": arenaServerUUID,
						},
					}),
				)

				if err != nil {
					utils.Debug("master", "ERROR: could not set game state to running for Game "+gameid+" on server "+arenaServerUUID)
				} else {
					utils.Debug("master", "Game state set to running for Game "+gameid+" on server "+arenaServerUUID)
				}
			}()
		} else {
			utils.Debug("master", "ERROR: arena ("+arenaServerUUID+") has been launched but wasn't in pending state")
		}
	} else {
		utils.Debug("master", "Received game launched event but payload is not parsable")
	}
}
