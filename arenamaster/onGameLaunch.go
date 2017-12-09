package arenamaster

import (
	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"

	"github.com/bytearena/core/common/types"
	"github.com/bytearena/core/common/utils"

	"github.com/bytearena/backends/common/graphql"
	"github.com/bytearena/backends/common/mq"
)

func onGameLaunch(gameid string, mqclient *mq.Client, gql *graphql.Client, vm *vm.VM) {

	vm.Config.Metadata["gameid"] = gameid
	mac, _ := vmid.GetVMMAC(vm)

	// TODO: should be wrapped in types.NewMQMessage
	mqclient.Publish("game", mac+".launch", types.MQPayload{
		"id": gameid,
	})

	utils.Debug("master", "Launched game "+gameid+" on server "+mac)
}
