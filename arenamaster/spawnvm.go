package arenamaster

import (
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"
	vmtypes "github.com/bytearena/schnapps/types"
)

var (
	MEG_MEMORY      = 2048
	CPU_AMOUNT      = 1
	CPU_CORE_AMOUNT = 1
)

func (server *Server) SpawnArena(id int) (*vm.VM, error) {
	mac := vmid.GenerateRandomMAC()
	ip, ipErr := server.DHCPServer.Pop()

	if ipErr != nil {
		return nil, ipErr
	}

	meta := vmtypes.VMMetadata{
		"IP": ip,
	}

	config := vmtypes.VMConfig{
		NICs: []interface{}{
			vmtypes.NICBridge{
				Bridge: server.vmBridgeName,
				MAC:    mac,
			},
		},
		Id:            id,
		MegMemory:     MEG_MEMORY,
		CPUAmount:     CPU_AMOUNT,
		CPUCoreAmount: CPU_CORE_AMOUNT,
		ImageLocation: server.vmRawImageLocation,
		Metadata:      meta,
	}

	arenaVm := vm.NewVM(config)

	startErr := arenaVm.Start()

	if startErr != nil {
		return nil, startErr
	}

	utils.Debug("vm", "Started new VM ("+mac+")")

	return arenaVm, nil
}
