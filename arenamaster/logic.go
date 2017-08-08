package arenamaster

import (
	"strconv"
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

const updateGameStateMutation = `
mutation ($id: String!, $game: GameInputUpdate!) {
	updateGame(id: $id, game: $game) {
		id
		runStatus
	}
}
`

func getMasterStatus(state *State) string {
	return "(" + strconv.Itoa(len(state.idleArenas)) + " arena(s) idle, " + strconv.Itoa(len(state.runningArenas)) + " arena(s) running)"
}

func onGameHandshake(state *State, payload *types.MQPayload) {
	id, ok := (*payload)["id"].(string)

	if ok {
		state.idleArenas[id] = ArenaState{
			id: id,
		}

		utils.Debug("master", id+" joined "+getMasterStatus(state))
	}
}

func onGameLaunch(state *State, payload *types.MQPayload, mqclient *mq.Client, gql *graphql.Client) {

	if len(state.idleArenas) > 0 {

		var astate ArenaState

		for _, value := range state.idleArenas {
			astate = value
			break
		}

		delete(state.idleArenas, astate.id)
		state.runningArenas[astate.id] = astate

		if id, ok := (*payload)["id"].(string); ok {

			mqclient.Publish("game", astate.id+".launch", types.MQPayload{
				"id": id,
			})

			utils.Debug("master", "Launched game "+astate.id+" "+getMasterStatus(state))

			go func() {
				_, err := gql.RequestSync(
					graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
						"id": id,
						"game": graphql.Variables{
							"runStatus":     gqltypes.GameRunStatus.Running,
							"launchedAt":    (time.Time{}).Format(time.RFC822Z),
							"arenaServerId": astate.id,
						},
					}),
				)

				if err != nil {
					utils.Debug("master", "ERROR: could not set game state to running for Game "+id+" on server "+astate.id)
				} else {
					utils.Debug("master", "Game state set to running for Game "+id+" on server "+astate.id)
				}
			}()
		}
	} else {
		utils.Debug("master", "No game available")
	}
}

func onGameStop(state *State, payload *types.MQPayload, gql *graphql.Client) {

	if id, ok := (*payload)["id"].(string); ok {
		arena, ok := state.runningArenas[id]

		if ok {
			delete(state.runningArenas, arena.id)

			utils.Debug("master", "Arena "+id+" stopped "+getMasterStatus(state))

			go func() {
				_, err := gql.RequestSync(
					graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
						"id": id,
						"game": graphql.Variables{
							"runStatus": gqltypes.GameRunStatus.Finished,
							"endedAt":   (time.Time{}).Format(time.RFC822Z),
						},
					}),
				)

				if err != nil {
					utils.Debug("master", "ERROR: could not set game state to finished for Game "+id)
				} else {
					utils.Debug("master", "Game state set to finished for Game "+id)
				}
			}()

		} else {
			utils.Debug("master", "Arena "+id+" is not running")
		}
	}
}
