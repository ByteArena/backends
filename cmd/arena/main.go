package main

import (
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bytearena/bytearena/common/api"
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/messagebroker"
	"github.com/bytearena/bytearena/server"
	"github.com/bytearena/bytearena/utils"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	arenahost, exists := os.LookupEnv("HOST") // override host if needed (useful for netgusto@macos)
	if !exists {
		arenahost = ""
	}

	// Arena PORT (for agents to connect to)
	arenaport, exists := os.LookupEnv("PORT")
	if !exists {
		arenaport = "8080"
	}
	arenaportint, _ := strconv.Atoi(arenaport)

	// Make message broker client
	brokerhost, exists := os.LookupEnv("MESSAGEBROKERHOST")
	utils.Assert(exists, "Error: MESSAGEBROKERHOST should be defined in the environment")

	brokerclient, err := messagebroker.NewClient(brokerhost)
	utils.Check(err, "ERROR: Could not connect to messagebroker on "+brokerhost)

	// Make GraphQL client
	apiendpoint, exists := os.LookupEnv("APIENDPOINT")
	utils.Assert(exists, "Error: APIENDPOINT should be defined in the environment")
	graphqlclient := graphql.MakeClient(apiendpoint)

	// Fetch arena **instance** from GraphQL
	arenainstanceid, exists := os.LookupEnv("ARENAINSTANCEID")
	if !exists {
		arenainstanceid = "1"
	}

	arena, err := api.FetchArenaInstanceById(graphqlclient, arenainstanceid)
	utils.Check(err, "Could not fetch arenainstance "+arenainstanceid)

	stopticking := make(chan bool)

	srv := server.NewServer(
		arena,
		arenahost, arenaportint,
		stopticking,
	)

	for _, contestant := range arena.GetContestants() {
		go srv.SpawnAgent(contestant.AgentRegistry + "/" + contestant.AgentImage)
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

	go streamState(srv, brokerclient)

	srv.Listen()
}
