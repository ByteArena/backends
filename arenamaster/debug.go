package arenamaster

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"

	"github.com/bytearena/backends/arenamaster/state"
	"github.com/bytearena/backends/common/mq"

	"github.com/bytearena/core/common/types"
)

func handleDebugGetVMStatus(mqClient *mq.Client, s *state.State, healthchecks *ArenaHealthCheck) {
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

		cache := healthchecks.GetCache()

		if res, hasRes := cache[mac]; hasRes {
			if res {
				debugState[id]["health"] = "OK"
			} else {
				debugState[id]["health"] = "NOK"
			}
		}

		lastSeen := healthchecks.GetLastSeen()

		if res, hasRes := lastSeen[mac]; hasRes {
			debugState[id]["lastseen"] = res.Format(time.RFC3339)
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
