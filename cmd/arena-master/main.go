package main

import (
	"os"

	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/healthcheck"

	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/utils"

	"github.com/bytearena/bytearena/arenamaster"
)

var (
	env                = os.Getenv("ENV")
	mqHost             = os.Getenv("MQ")
	apiUrl             = os.Getenv("APIURL")
	vmRawImageLocation = utils.GetenvOrDefault("VM_RAW_IMAGE_LOCATION", "/linuxkit.raw")
)

func main() {
	utils.Assert(mqHost != "", "MQ must be set")
	utils.Assert(apiUrl != "", "APIURL must be set")

	brokerclient, err := mq.NewClient(mqHost)
	utils.Check(err, "ERROR: could not connect to messagebroker at "+string(mqHost))

	graphqlclient := graphql.NewClient(apiUrl)

	server := arenamaster.NewServer(brokerclient, graphqlclient, vmRawImageLocation)

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
