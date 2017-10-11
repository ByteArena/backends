package arenamaster

import (
	"github.com/bytearena/bytearena/arenamaster/state"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/schnapps"
)

func handleDebugGetVMStatus(mqClient *mq.Client, s *state.State) {
	debugState := make(map[int][]string)

	s.Map(func(element *state.DataContainer) {
		vm := element.Data.(*vm.VM)
		id := vm.Config.Id

		debugState[id] = s.DebugGetStatus(id)
	})

	mqClient.Publish("debug", "getvmstatus-res", types.MQPayload{
		"state": debugState,
	})
}
