package arenamaster

import (
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

const updateGameStateMutation = `
mutation ($id: String, $arenaServerId: String, $game: GameInputUpdate!) {
	updateGame(id: $id, arenaServerId: $arenaServerId, game: $game) {
		id
		runStatus
	}
}
`

func onGameStop(state *State, payload *types.MQPayload, gql *graphql.Client) {

	if arenaServerId, ok := (*payload)["arenaserverid"].(string); ok {
		arena, ok := state.runningArenas[arenaServerId]

		if ok {
			delete(state.runningArenas, arena.id)

			utils.Debug("master", "Arena running on server "+arenaServerId+" stopped "+getMasterStatus(state))

			go func() {
				_, err := gql.RequestSync(
					graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
						"arenaServerId": arenaServerId,
						"game": graphql.Variables{
							"runStatus": gqltypes.GameRunStatus.Finished,
							"endedAt":   time.Now().Format(time.RFC822Z),
						},
					}),
				)

				if err != nil {
					utils.Debug("master", "ERROR: could not set game state to finished for Game running on arena server "+arenaServerId)
				} else {
					utils.Debug("master", "Game state set to finished for Game running on arena server "+arenaServerId)
				}
			}()

		} else {
			utils.Debug("master", "Arena server "+arenaServerId+" is not running an arena")
		}
	}
}
