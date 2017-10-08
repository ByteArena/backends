package arenamaster

import (
	"strings"

	"github.com/bytearena/bytearena/arenamaster/vm"
	"github.com/bytearena/bytearena/arenamaster/vm/types"
)

func FindVMByMAC(state *State, mac string) *vm.VM {
	upperMac := strings.ToUpper(mac)

	for _, element := range state.state {
		if vm, isVm := element.Data.(*vm.VM); isVm {

			for _, nic := range vm.Config.NICs {
				if bridge, ok := nic.(types.NICBridge); ok {
					if bridge.MAC == upperMac {
						return vm
					}

				}
			}

		}
	}

	return nil
}
