package arenamaster

import (
	"github.com/bytearena/bytearena/common/utils"
	vmdhcp "github.com/bytearena/schnapps/dhcp"
)

func (server *Server) createDHCPServer() {
	var err error
	cidr := server.vmSubnet

	server.DHCPServer, err = vmdhcp.NewDHCPServer(cidr)
	utils.Check(err, "Could not create DHCP server")
}
