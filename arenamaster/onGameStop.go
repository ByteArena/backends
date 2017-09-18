package arenamaster

import (
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

const updateGameStateMutation = `
updateGame(id: $id, game: $game) {
	mutation ($id: String, $game: GameInputUpdate!) {
		id
		runStatus
	}
}
`

func onGameStop(state *State, payload *types.MQPayload, gql *graphql.Client) {

	if arenaServerUUID, ok := (*payload)["arenaserveruuid"].(string); ok {

		state.LockState()
		defer state.UnlockState()

		arena, ok := state.runningArenas[arenaServerUUID]

		if ok {
			delete(state.runningArenas, arena.id)

			gameid, _ := (*payload)["id"].(string)

			utils.Debug("master", "Game "+gameid+" running on server "+arenaServerUUID+" stopped "+getMasterStatus(state))

			go func() {
				_, err := gql.RequestSync(
					graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
						"id": gameid,
						"game": graphql.Variables{
							"runStatus":       gqltypes.GameRunStatus.Finished,
							"endedAt":         time.Now().Format(time.RFC822Z),
							"arenaServerUUID": arenaServerUUID,
						},
					}),
				)

				if err != nil {
					utils.Debug("master", "ERROR: could not set game state to finished for Game "+gameid+" running on arena server "+arenaServerUUID)
				} else {
					utils.Debug("master", "Game state set to finished for Game  "+gameid+" running on arena server "+arenaServerUUID)
				}
			}()

		} else {
			utils.Debug("master", "Arena ("+arenaServerUUID+") was stopped but was not in the running state")
		}
	}
}
