package queries

import (
	"encoding/json"
	"log"

	"errors"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common/graphql"
	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
)

const arenainstanceQuery = `
query ($instanceid: String = null) {
	arenainstances(id: $instanceid) {
		id
		startedat
		endedat
		tps
		arena {
			id
			name
			kind
			maxcontestants
			surface { width, height }
			obstacles {
				a { x, y }
				b { x, y }
			}
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

func FetchArenaInstances(graphqlclient graphql.Client) ([]arenaserver.ArenaInstance, error) {
	data, err := graphqlclient.RequestSync(
		graphql.NewQuery(arenainstanceQuery),
	)

	if err != nil {
		return nil, errors.New("Could not fetch arena instances from GraphQL")
	}

	var apiresponse struct {
		Arenainstances []graphqltype.ArenaInstanceType `json:"arenainstances"`
	}
	json.Unmarshal(data, &apiresponse)

	res := make([]arenaserver.ArenaInstance, 0)
	for _, arenainstance := range apiresponse.Arenainstances {
		res = append(res, arenaserver.NewArenaInstanceGql(arenainstance))
	}

	return res, nil
}

func FetchArenaInstanceById(graphqlclient graphql.Client, arenainstanceid string) (arenaserver.ArenaInstance, error) {

	data, err := graphqlclient.RequestSync(
		graphql.NewQuery(arenainstanceQuery).SetVariables(graphql.Variables{
			"instanceid": arenainstanceid,
		}),
	)

	if err != nil {
		log.Panicln(err)
		return nil, errors.New("Could not fetch arena instance '" + arenainstanceid + "' from GraphQL")
	}

	var apiresponse struct {
		Arenainstances []graphqltype.ArenaInstanceType `json:"arenainstances"`
	}
	json.Unmarshal(data, &apiresponse)
	arena := arenaserver.NewArenaInstanceGql(apiresponse.Arenainstances[0])

	return arena, nil
}
