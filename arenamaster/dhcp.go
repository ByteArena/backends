package arenamaster

import (
	vmdhcp "github.com/bytearena/schnapps/dhcp"

	"github.com/bytearena/core/common/utils"
)

func (server *Server) createDHCPServer() {
	var err error
	cidr := server.vmSubnet

	server.DHCPServer, err = vmdhcp.NewDHCPServer(cidr)
	utils.Check(err, "Could not create DHCP server")
}
