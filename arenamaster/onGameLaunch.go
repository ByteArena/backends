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

	if gameid, ok := (*payload)["id"].(string); ok {
		state.LockState()

		if len(state.idleArenas) > 0 {

			// Ignore if the game is already running
			if isGameAlreadyRunning(state, gameid) {
				state.UnlockState()

				utils.Debug("master", "ERROR: game "+gameid+" is already running "+getMasterStatus(state))
				return
			}

			var astate ArenaServerState

			// Take the first arena
			for _, value := range state.idleArenas {
				astate = value
				break
			}

			// Remove from idle pool
			delete(state.idleArenas, astate.id)

			astate.GameId = gameid

			// Put it into pending arenas (waiting for arena comfirmation)
			state.pendingArenas[astate.id] = astate

			state.UnlockState()

			// TODO: should be wrapped in types.NewMQMessage
			mqclient.Publish("game", astate.id+".launch", types.MQPayload{
				"id": gameid,
			})

			utils.Debug("master", "Launched game "+gameid+" on server "+astate.id+"; "+getMasterStatus(state))

			go func() {
				_, err := gql.RequestSync(
					graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
						"id": gameid,
						"game": graphql.Variables{
							"runStatus":       gqltypes.GameRunStatus.Running,
							"launchedAt":      time.Now().Format(time.RFC822Z),
							"arenaServerUUID": astate.id,
						},
					}),
				)

				if err != nil {
					utils.Debug("master", "ERROR: could not set game state to running for Game "+gameid+" on server "+astate.id)
				} else {
					utils.Debug("master", "Game state set to running for Game "+gameid+" on server "+astate.id)
				}
			}()

			go waitForLaunchedOrRetry(state, payload, mqclient, gql, astate)
		} else {
			state.UnlockState()
			utils.Debug("master", "No arena available for game "+gameid)
		}
	} else {
		utils.Debug("master", "Received game launch event but payload is not parsable")
	}
}

func waitForLaunchedOrRetry(state *State, payload *types.MQPayload, mqclient *mq.Client, gql *graphql.Client, astate ArenaServerState) {
	timeout := 30
	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
	<-timeoutTimer.C

	state.LockState()

	_, isPending := state.pendingArenas[astate.id]

	if isPending {
		utils.Debug("pending", "Arena "+astate.id+" couldn't be launched")

		delete(state.pendingArenas, astate.id)
		state.UnlockState()

		// Retry to launch a game
		onGameStop(state, payload, gql)
		onGameLaunch(state, payload, mqclient, gql)
	} else {
		state.UnlockState()
	}
}

func isGameAlreadyRunning(state *State, id string) bool {
	// FIXME(sven): missing check in pending arenas

	for _, a := range state.runningArenas {
		if a.GameId == id {
			return true
		}
	}

	return false
}
