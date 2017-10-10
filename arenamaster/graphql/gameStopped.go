package graphql

import (
	"time"

	"github.com/bytearena/bytearena/arenamaster/state"
	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/utils"
)

const updateGameStateMutation = `
mutation($id: String, $game: GameInputUpdate!) {
	updateGame(id: $id, game: $game) {
		id
		runStatus
	}
}
`

func ReportGameStopped(state *state.State, arenaServerUUID, gameid string, gql *graphql.Client) {

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

}