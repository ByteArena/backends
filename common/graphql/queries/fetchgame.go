package queries

import (
	"encoding/json"

	"errors"

	"github.com/bytearena/backends/common/graphql"
	graphqltype "github.com/bytearena/backends/common/graphql/types"
	"github.com/bytearena/core/common/types"
)

const gameQuery = `
query ($gameid: String = null) {
	games(id: $gameid) {
		id
		launchedAt
		endedAt
		tps
		arena {
			id
			name
			kind
			maxContestants
		}
		contestants {
			id
			agent {
				id
				name
				owner {
					id
					name
					username
				}
				image {
					name
					tag
					registry
				}
			}
		}
	}
}
`

func FetchGames(graphqlclient graphql.Client) ([]types.GameDescriptionInterface, error) {
	data, err := graphqlclient.RequestSync(
		graphql.NewQuery(gameQuery),
	)

	if err != nil {
		return nil, errors.New("Could not fetch games from GraphQL")
	}

	var apiresponse struct {
		Games []graphqltype.GameType `json:"games"`
	}
	json.Unmarshal(data, &apiresponse)

	res := make([]types.GameDescriptionInterface, 0)
	for _, game := range apiresponse.Games {
		res = append(res, types.NewGameDescriptionGQL(game))
	}

	return res, nil
}

func FetchGameById(graphqlclient graphql.Client, gameid string) (types.GameDescriptionInterface, error) {

	data, err := graphqlclient.RequestSync(
		graphql.NewQuery(gameQuery).SetVariables(graphql.Variables{
			"gameid": gameid,
		}),
	)

	if err != nil {
		return nil, errors.New("Could not fetch game '" + gameid + "' from GraphQL")
	}

	var apiresponse struct {
		Games []graphqltype.GameType `json:"games"`
	}
	json.Unmarshal(data, &apiresponse)
	game := types.NewGameDescriptionGQL(apiresponse.Games[0])

	return game, nil
}
