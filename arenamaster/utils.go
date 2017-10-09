package arenamaster

import (
	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"
)

func FindVMByMAC(state *State, searchMac string) *vm.VM {

	for _, element := range state.state {
		if vm, isVm := element.Data.(*vm.VM); isVm {
			mac, found := vmid.GetVMMAC(vm)

			if searchMac == mac && found {
				return vm
			}

		}
	}

	return nil
}
