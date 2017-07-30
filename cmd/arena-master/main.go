package main

import (
	"os"
	"os/signal"
	"syscall"

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

	if env == "prod" {
		StartHealthCheck(brokerclient)
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		server.Stop()
	}()

	<-server.Start()
}
