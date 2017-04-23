package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netgusto/bytearena/server"
	"github.com/netgusto/bytearena/server/config"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils/vector"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	filename := os.Args[1]
	config := config.LoadServerConfig(filename)

	host, exists := os.LookupEnv("HOST")
	if !exists {
		host = ""
	}

	stopticking := make(chan bool)

	srv := server.NewServer(
		host,
		config.Port,
		len(config.Agents),
		config.Tps,
		stopticking,
	)

	// Creating obstacles

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(0, 0),
		vector.MakeVector2(1000, 0),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(1000, 0),
		vector.MakeVector2(1000, 600),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(1000, 600),
		vector.MakeVector2(0, 600),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(0, 600),
		vector.MakeVector2(0, 0),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(100, 100),
		vector.MakeVector2(900, 100),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(900, 100),
		vector.MakeVector2(900, 500),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(900, 500),
		vector.MakeVector2(500, 500),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(100, 500),
		vector.MakeVector2(100, 100),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(300, 300),
		vector.MakeVector2(300, 500),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(700, 200),
		vector.MakeVector2(500, 400),
	))

	for _, agentconfig := range config.Agents {
		go srv.Spawnagent(agentconfig)
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
