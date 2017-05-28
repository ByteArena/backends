package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bytearena/bytearena/server"
	"github.com/bytearena/bytearena/server/config"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	filename := os.Args[1]
	config := config.LoadServerConfig(filename)

	host, exists := os.LookupEnv("HOST")
	if !exists {
		host = ""
	}

	arena := server.NewSandboxArena()
	stopticking := make(chan bool)

	srv := server.NewServer(
		host,
		config.Port,
		len(config.Agents),
		config.Tps,
		stopticking,
		arena,
	)

	// Spawn agents

	for _, agentconfig := range config.Agents {
		go srv.SpawnAgent(agentconfig)
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		stopticking <- true
		srv.TearDown()
		os.Exit(1)
	}()

	go visualization(srv, "0.0.0.0", config.Port+1)

	srv.Listen()
}
