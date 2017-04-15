package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netgusto/bytearena/agents/attractor"
	"github.com/netgusto/bytearena/server"
	"github.com/netgusto/bytearena/server/config"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	filename := os.Args[1]
	config := config.LoadServerConfig(filename)

	stopticking := make(chan bool)

	srv := server.NewServer(
		config.Host,
		config.Port,
		len(config.Agents),
		config.Tps,
		stopticking,
	)

	// Creating attractor as an agent
	agentstate := state.MakeAgentState()
	agentstate.Tag = "attractor"
	agentstate.Position = utils.MakeVector2(400, 300)
	agentstate.Radius = 16
	srv.RegisterAgent(attractoragent.MakeAttractorAgent(), agentstate)

	for _, agentconfig := range config.Agents {
		go srv.Spawnagent(config.Agentdir, agentconfig)
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

	go visualization(srv, config.Host, config.Port+1)

	srv.Listen()
}
