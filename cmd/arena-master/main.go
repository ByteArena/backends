package main

import (
	"os"

	"github.com/bytearena/bytearena/common/healthcheck"

	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/utils"

	"github.com/bytearena/bytearena/arenamaster"
)

func main() {
	env := os.Getenv("ENV")
	mqHost := os.Getenv("MQ")

	utils.Assert(mqHost != "", "MQ must be set")

	brokerclient, err := mq.NewClient(mqHost)
	utils.Check(err, "ERROR: could not connect to messagebroker at "+string(mqHost))

	server := arenamaster.NewServer(brokerclient)

	// handling signals
	var hc *healthcheck.HealthCheckServer
	if env == "prod" {
		hc = NewHealthCheck(brokerclient)
		hc.Start()
	}

	go func() {

		<-common.SignalHandler()
		utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")

		server.Stop()

		if hc != nil {
			hc.Stop()
		}
	}()

	<-server.Start()
}
