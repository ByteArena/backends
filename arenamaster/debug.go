package arenamaster

import (
	"encoding/json"
	"strings"

	"github.com/bytearena/bytearena/arenamaster/state"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"
)

func handleDebugGetVMStatus(mqClient *mq.Client, s *state.State) {
	debugState := make(map[int]map[string]string)

	s.Map(func(element *state.DataContainer) {
		vm := element.Data.(*vm.VM)
		id := vm.Config.Id

		debugState[id] = make(map[string]string)
		debugState[id]["state"] = strings.Join(s.DebugGetStatus(id), ",")

		mac, found := vmid.GetVMMAC(vm)

		if found {
			debugState[id]["mac"] = mac
		}

		metadatajson, err := json.Marshal(vm.Config.Metadata)

		if err != nil {
			debugState[id]["metadata"] = err.Error()
		} else {
			debugState[id]["metadata"] = string(metadatajson)
		}
	})

	mqClient.Publish("debug", "getvmstatus-res", types.MQPayload{
		"state": debugState,
	})
}
