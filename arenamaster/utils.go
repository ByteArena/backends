package arenamaster

import (
	"github.com/bytearena/bytearena/arenamaster/vm"
	"github.com/bytearena/bytearena/arenamaster/vm/types"
)

func GetVMMAC(vm *vm.VM) (mac string, found bool) {

	for _, nic := range vm.Config.NICs {
		if bridge, ok := nic.(types.NICBridge); ok {
			return bridge.MAC, true
		}
	}

	return "", false
}

func FindVMByMAC(state *State, searchMac string) *vm.VM {

	for _, element := range state.state {
		if vm, isVm := element.Data.(*vm.VM); isVm {
			mac, found := GetVMMAC(vm)

			if searchMac == mac && found {
				return vm
			}

		}
	}

	return nil
}
