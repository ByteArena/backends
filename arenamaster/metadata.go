package arenamaster

import (
	"fmt"

	"github.com/bytearena/schnapps"
	vmmeta "github.com/bytearena/schnapps/metadata"

	"github.com/bytearena/core/common/utils"
)

var (
	PORT = 8080
)

func (server *Server) createMetadataServer() {
	retrieveVMFn := func(id string) *vm.VM {
		vm := FindVMByMAC(server.state, id)

		return vm
	}

	metadataServer := vmmeta.NewServer(fmt.Sprintf("%s:%d", server.vmBridgeIP, PORT), retrieveVMFn)

	go func() {
		err := metadataServer.Start()
		utils.Check(err, "Could not start metadata server")

		server.MetadataServer = metadataServer
	}()
}
