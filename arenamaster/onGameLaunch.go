package arenamaster

import (
	"time"

	"github.com/bytearena/bytearena/arenamaster/vm"
	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func onGameLaunch(gameid string, mqclient *mq.Client, gql *graphql.Client, vm *vm.VM) {

	// FIXME(sven) re-enable already running game id protection
	// Ignore if the game is already running
	// if isGameAlreadyRunning(state, gameid) {
	// 	state.UnlockState()

	// 	utils.Debug("master", "ERROR: game "+gameid+" is already running "+getMasterStatus(state))
	// 	return
	// }

	vm.Gameid = gameid
	mac, _ := GetVMMAC(vm)

	// TODO: should be wrapped in types.NewMQMessage
	mqclient.Publish("game", mac+".launch", types.MQPayload{
		"id": gameid,
	})

	utils.Debug("master", "Launched game "+gameid+" on server "+mac)

	go func() {
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
	}()

	// go waitForLaunchedOrRetry(state, gameid, mqclient, gql, astate, vm)
}

// FIXME(sven) re-enable retry mecasim
// func waitForLaunchedOrRetry(state *State, gameid string, mqclient *mq.Client, gql *graphql.Client, astate ArenaServerState, vm *vm.VM) {
// 	timeout := 30
// 	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
// 	<-timeoutTimer.C

// 	state.LockState()

// 	_, isPending := state.pendingArenas[astate.id]

// 	if isPending {
// 		utils.Debug("pending", "Arena "+astate.id+" couldn't be launched")

// 		delete(state.pendingArenas, astate.id)
// 		state.UnlockState()

// 		// Retry to launch a game
// 		// TODO(sven): stop only if needed
// 		// onGameStop(state, "?", gameid, gql)
// 		onGameLaunch(state, gameid, mqclient, gql, vm)
// 	} else {
// 		state.UnlockState()
// 	}
// }

// func isGameAlreadyRunning(state *State, id string) bool {
// 	for _, a := range state.runningArenas {
// 		if a.GameId == id {
// 			return true
// 		}
// 	}

// 	return false
// }
