package arenamaster

import (
	"github.com/bytearena/bytearena/common/utils"
	vmdns "github.com/bytearena/schnapps/dns"
)

var (
	dnsZone = "bytearena.com."
)

func (server *Server) createDNSServer() {

	dnsRecords := map[string]string{
		"static." + dnsZone:       server.vmBridgeIP,
		"redis.net." + dnsZone:    server.vmBridgeIP,
		"graphql.net." + dnsZone:  server.vmBridgeIP,
		"registry.net." + dnsZone: server.vmBridgeIP,
	}

	DNSServer := vmdns.MakeServer(server.vmBridgeIP+":53", dnsZone, dnsRecords)

	// DNSServer.SetOnRequestHook(func(addr string) {
	// 	utils.Debug("dns-server", "query for "+addr)
	// })

	go func() {
		err := DNSServer.Start()
		utils.Check(err, "Could not start DNS server")

		server.DNSServer = &DNSServer
	}()
}
