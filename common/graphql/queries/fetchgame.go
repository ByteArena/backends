package queries

import (
	"encoding/json"
	"log"

	"errors"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common/graphql"
	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
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

func FetchGames(graphqlclient graphql.Client) ([]arenaserver.GameInterface, error) {
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

	res := make([]arenaserver.GameInterface, 0)
	for _, game := range apiresponse.Games {
		res = append(res, arenaserver.NewGameGql(game))
	}

	return res, nil
}

func FetchGameById(graphqlclient graphql.Client, gameid string) (arenaserver.GameInterface, error) {

	data, err := graphqlclient.RequestSync(
		graphql.NewQuery(gameQuery).SetVariables(graphql.Variables{
			"gameid": gameid,
		}),
	)

	if err != nil {
		log.Panicln(err)
		return nil, errors.New("Could not fetch game '" + gameid + "' from GraphQL")
	}

	var apiresponse struct {
		Games []graphqltype.GameType `json:"games"`
	}
	json.Unmarshal(data, &apiresponse)
	arena := arenaserver.NewGameGql(apiresponse.Games[0])

	return arena, nil
}
