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
mutation ($id: String, $arenaServerId: String, $game: GameInputUpdate!) {
	updateGame(id: $id, arenaServerId: $arenaServerId, game: $game) {
		id
		runStatus
	}
}
`

func getMasterStatus(state *State) string {
	return "(" + strconv.Itoa(len(state.idleArenas)) + " arena(s) idle, " + strconv.Itoa(len(state.runningArenas)) + " arena(s) running, " + strconv.Itoa(len(state.pendingArenas)) + " arena(s) pending)"
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

		if id, ok := (*payload)["id"].(string); ok {

			var astate ArenaState

			// Take the first arena
			for _, value := range state.idleArenas {
				astate = value
				break
			}

			// Remove from idle pool
			delete(state.idleArenas, astate.id)

			// Put it into pending arenas (waiting for arena comfirmation)
			state.pendingArenas[astate.id] = astate

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
							"launchedAt":    time.Now().Format(time.RFC822Z),
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

			go func() {
				timeout := 30
				timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
				<-timeoutTimer.C

				_, isPending := state.pendingArenas[astate.id]

				if isPending {
					utils.Debug("pending", "Arena "+astate.id+" couldn't be launched")

					delete(state.pendingArenas, astate.id)

					// Retry to launch a game
					onGameLaunch(state, payload, mqclient, gql)
				}
			}()
		}
	} else {
		utils.Debug("master", "No game available")
	}
}

func onGameLaunched(state *State, payload *types.MQPayload, mqclient *mq.Client, gql *graphql.Client) {
	if arenaServerId, ok := (*payload)["arenaserverid"].(string); ok {
		if arena, ok := state.pendingArenas[arenaServerId]; ok {

			// Put it into running arenas, now that we're sure
			state.runningArenas[arena.id] = arena

			delete(state.pendingArenas, arena.id)

			utils.Debug("master", arenaServerId+" launched "+getMasterStatus(state))
		}
	}
}

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
