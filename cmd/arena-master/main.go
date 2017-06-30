package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/utils"

	"github.com/bytearena/bytearena/arenamaster"
)

func GetConfig() (mqHost string, err error) {
	mqHostEnv := os.Getenv("MQ")
	utils.Assert(mqHostEnv != "", "MQ must be set")

	return mqHostEnv, nil
}

func main() {
	mqHost, err := GetConfig()
	utils.Check(err, "ERROR: could not get config")

	brokerclient, err := mq.NewClient(mqHost)
	utils.Check(err, "ERROR: could not connect to messagebroker at "+string(mqHost))

	server := arenamaster.NewServer(brokerclient)

	StartHealthCheck(brokerclient)

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		server.Stop()
	}()

	<-server.Start()
}
