package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bytearena/bytearena/common/api"
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/messagebroker"
	commonprotocol "github.com/bytearena/bytearena/common/protocol"
	"github.com/bytearena/bytearena/server"
	"github.com/bytearena/bytearena/utils"
)

func main() {

	rand.Seed(time.Now().UnixNano())
	log.Println("Byte Arena Trainer v0.1")

	host := flag.String("host", "", "IP serving the arena; required")
	port := flag.Int("port", 8080, "Port serving the arena")
	mqhost := flag.String("mqhost", "mq:5678", "Message queue host:port")
	apiurl := flag.String("apiurl", "http://bytearena.com/privateapi/graphql", "GQL API URL")
	arenainstanceid := flag.String("arenainstance", "", "Arena instance id")

	flag.Parse()

	if *arenainstanceid == "" {
		fmt.Println("-arenainstance is required")
		os.Exit(1)
	}

	// Make message broker client
	brokerclient, err := messagebroker.NewClient(*mqhost)
	utils.Check(err, "ERROR: Could not connect to messagebroker on "+*mqhost)

	// Make GraphQL client
	graphqlclient := graphql.MakeClient(*apiurl)

	// Fetch arena **instance** from GraphQL
	arena, err := api.FetchArenaInstanceById(graphqlclient, *arenainstanceid)
	utils.Check(err, "Could not fetch arenainstance "+*arenainstanceid)
	log.Println(arena)

	srv := server.NewServer(*host, *port, arena)

	for _, contestant := range arena.GetContestants() {
		srv.RegisterAgent(contestant.AgentRegistry + "/" + contestant.AgentImage)
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		srv.Stop()
	}()

	go commonprotocol.StreamState(srv, brokerclient)

	<-srv.Start()
	srv.TearDown()
}
