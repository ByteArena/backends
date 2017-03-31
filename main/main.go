package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/kardianos/osext"
	"github.com/netgusto/bytearena/server"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	host := os.Getenv("HOST")

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		port = 8080
	}

	nbagents, err := strconv.Atoi(os.Getenv("AGENTS"))
	if err != nil {
		nbagents = 8
	}

	tickspersec, err := strconv.Atoi(os.Getenv("TPS"))
	if err != nil {
		tickspersec = 10
	}

	ctx := context.Background()

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	stopticking := make(chan bool)

	swarm := server.NewSwarm(
		ctx,
		host,
		port,
		exfolder+"/../agents/seeker",
		nbagents,
		tickspersec,
		stopticking,
	)

	for i := 0; i < nbagents; i++ {
		go swarm.Spawnagent()
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		stopticking <- true
		swarm.Teardown()
		os.Exit(1)
	}()

	swarm.Listen()
}
