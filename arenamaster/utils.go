package arenamaster

import (
	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"

	"github.com/bytearena/backends/arenamaster/state"
)

func FindVMByMAC(s *state.State, searchMac string) *vm.VM {
	var res *vm.VM

	s.Map(func(element *state.DataContainer) {
		if vm, isVm := element.Data.(*vm.VM); isVm {
			mac, found := vmid.GetVMMAC(vm)

			if searchMac == mac && found {
				res = vm
				return
			}
		}
	})

	return res
}
