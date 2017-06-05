package api

import (
	"encoding/json"
	"log"

	"errors"

	"github.com/bytearena/bytearena/common/graphql"
	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/server"
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
				repo
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

func FetchArenaInstances(graphqlclient graphql.Client) ([]server.ArenaInstance, error) {
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

	res := make([]server.ArenaInstance, 0)
	for _, arenainstance := range apiresponse.Arenainstances {
		res = append(res, server.NewArenaInstanceGql(arenainstance))
	}

	return res, nil
}

func FetchArenaInstanceById(graphqlclient graphql.Client, arenainstanceid string) (server.ArenaInstance, error) {

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
	arena := server.NewArenaInstanceGql(apiresponse.Arenainstances[0])

	return arena, nil
}
