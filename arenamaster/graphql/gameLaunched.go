package graphql

import (
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/utils"
)

func ReportGameLaunched(gameid, mac string, gql *graphql.Client) {

	// syncing state in graphql db
	_, err := gql.RequestSync(
		graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
			"id": gameid,
			"game": graphql.Variables{
				"runStatus":       gqltypes.GameRunStatus.Running,
				"launchedAt":      time.Now().Format(time.RFC822Z),
				"arenaServerUUID": mac,
			},
		}),
	)

	if err != nil {
		utils.Debug("master", "ERROR: could not set game state to running for Game "+gameid+" on server "+mac)
	} else {
		utils.Debug("master", "Game state set to running for Game "+gameid+" on server "+mac)
	}
}
