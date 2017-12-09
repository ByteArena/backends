package main

import (
	"os"

	"github.com/bytearena/backends/arenamaster"
	"github.com/bytearena/backends/common/graphql"
	"github.com/bytearena/backends/common/healthcheck"
	"github.com/bytearena/backends/common/mq"

	"github.com/bytearena/core/common"
	"github.com/bytearena/core/common/utils"
)

var (
	env                = os.Getenv("ENV")
	mqHost             = os.Getenv("MQ")
	apiUrl             = os.Getenv("APIURL")
	vmRawImageLocation = utils.GetenvOrDefault("VM_RAW_IMAGE_LOCATION", "/linuxkit.raw")
	vmBridgeName       = utils.GetenvOrDefault("VM_BRIDGE_NAME", "brtest")
	vmBridgeIP         = utils.GetenvOrDefault("VM_BRIDGE_IP", "172.19.0.1")
	vmSubnet           = utils.GetenvOrDefault("VM_SUBNET", "172.19.0.10/24")
)

func main() {
	utils.Assert(mqHost != "", "MQ must be set")
	utils.Assert(apiUrl != "", "APIURL must be set")

	brokerclient, err := mq.NewClient(mqHost)
	utils.Check(err, "ERROR: could not connect to messagebroker at "+string(mqHost))

	graphqlclient := graphql.NewClient(apiUrl)

	server := arenamaster.NewServer(brokerclient, graphqlclient, vmRawImageLocation, vmBridgeName, vmBridgeIP, vmSubnet)

	// handling signals
	var hc *healthcheck.HealthCheckServer
	if env == "prod" {
		hc = NewHealthCheck(brokerclient, graphqlclient)
		hc.Start()
	}

	go func() {
		<-common.SignalHandler()
		utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")

		server.Stop()
		brokerclient.Stop()

		if hc != nil {
			hc.Stop()
		}
	}()

	server.Run()
}
