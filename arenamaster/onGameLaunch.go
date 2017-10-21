package arenamaster

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"
)

func onGameLaunch(gameid string, mqclient *mq.Client, gql *graphql.Client, vm *vm.VM) {

	vm.Config.Metadata["gameid"] = gameid
	mac, _ := vmid.GetVMMAC(vm)

	// TODO: should be wrapped in types.NewMQMessage
	mqclient.Publish("game", mac+".launch", types.MQPayload{
		"id": gameid,
	})

	utils.Debug("master", "Launched game "+gameid+" on server "+mac)

	// go func() {
	// 	_, err := gql.RequestSync(
	// 		graphql.NewQuery(updateGameStateMutation).SetVariables(graphql.Variables{
	// 			"id": gameid,
	// 			"game": graphql.Variables{
	// 				"runStatus":       gqltypes.GameRunStatus.Running,
	// 				"launchedAt":      time.Now().Format(time.RFC822Z),
	// 				"arenaServerUUID": mac,
	// 			},
	// 		}),
	// 	)

	// 	if err != nil {
	// 		utils.Debug("master", "ERROR: could not set game state to running for Game "+gameid+" on server "+mac)
	// 	} else {
	// 		utils.Debug("master", "Game state set to running for Game "+gameid+" on server "+mac)
	// 	}
	// }()

	// go waitForLaunchedOrRetry(state, gameid, mqclient, gql, astate, vm)
}
