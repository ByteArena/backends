package arenamaster

import (
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func onGameLaunch(state *State, payload *types.MQPayload, mqclient *mq.Client, gql *graphql.Client) {

	if len(state.idleArenas) > 0 {

		if id, ok := (*payload)["id"].(string); ok {

			// Ignore if the game is already running
			if isGameIdAlreadyRunning(state, id) {
				return
			}

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

			go waitForLaunchedOrRetry(state, payload, mqclient, gql, astate)
		}
	} else {
		utils.Debug("master", "No game available")
	}
}

func waitForLaunchedOrRetry(state *State, payload *types.MQPayload, mqclient *mq.Client, gql *graphql.Client, astate ArenaState) {
	timeout := 30
	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
	<-timeoutTimer.C

	_, isPending := state.pendingArenas[astate.id]

	if isPending {
		utils.Debug("pending", "Arena "+astate.id+" couldn't be launched")

		delete(state.pendingArenas, astate.id)

		// Retry to launch a game
		onGameStop(state, payload, gql)
		onGameLaunch(state, payload, mqclient, gql)
	}
}

func isGameIdAlreadyRunning(state *State, id string) bool {
	// FIXME(sven): missing check in pending arenas

	for _, a := range state.runningArenas {
		if a.GameId == id {
			return true
		}
	}

	return false
}
