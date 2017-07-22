package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common/graphql"
	apiqueries "github.com/bytearena/bytearena/common/graphql/queries"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/protocol"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	uuid "github.com/satori/go.uuid"
)

type messageArenaLaunch struct {
	Id string `json:"id"`
}

func main() {

	rand.Seed(time.Now().UnixNano())
	arenaServerUUID := uuid.NewV4()
	log.Println("Byte Arena Server v0.1 ID#" + arenaServerUUID.String())

	host := flag.String("host", "", "IP serving the arena; required")
	port := flag.Int("port", 8080, "Port serving the arena")
	mqhost := flag.String("mqhost", "mq:5678", "Message queue host:port")
	apiurl := flag.String("apiurl", "http://bytearena.com/privateapi/graphql", "GQL API URL")

	flag.Parse()

	// Make GraphQL client
	graphqlclient := graphql.MakeClient(*apiurl)

	// Make message broker client
	brokerclient, err := mq.NewClient(*mqhost)
	utils.Check(err, "ERROR: Could not connect to messagebroker on "+*mqhost)

	brokerclient.Publish("arena", "handshake", types.NewMQMessage(
		"arena-server",
		"Arena Server "+arenaServerUUID.String()+" reporting for duty.",
	).SetPayload(types.MQPayload{
		"id": arenaServerUUID.String(),
	}))

	streamArenaLaunched := make(chan interface{})
	notify.Start("arena:launch", streamArenaLaunched)

	brokerclient.Subscribe("arena", arenaServerUUID.String()+".launch", func(msg mq.BrokerMessage) {

		var payload messageArenaLaunch
		err := json.Unmarshal(msg.Data, &payload)
		if err != nil {
			log.Println(err)
			log.Println("ERROR:arena:launch Invalid payload " + string(msg.Data))
			return
		}

		log.Println("INFO:arena:launch Received from MESSAGEBROKER", payload)

		notify.PostTimeout("arena:launch", payload, time.Millisecond)
	})

	StartHealthCheck(brokerclient, graphqlclient)

	go func() {
		for {
			select {
			case payload := <-streamArenaLaunched:
				{
					if arenaSubmitted, ok := payload.(messageArenaLaunch); ok {

						// Fetch arena **instance** from GraphQL
						arena, err := apiqueries.FetchArenaInstanceById(graphqlclient, arenaSubmitted.Id)
						utils.Check(err, "Could not fetch arenainstance "+arenaSubmitted.Id)

						srv := arenaserver.NewServer(*host, *port, container.MakeRemoteContainerOrchestrator(), arena)

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

						go protocol.StreamState(srv, brokerclient)

						<-srv.Start()
						srv.TearDown()

						notify.PostTimeout("arena:stopped", nil, time.Millisecond)
					}
				}
			}
		}
	}()

	streamArenaStopped := make(chan interface{})
	notify.Start("arena:stop", streamArenaStopped)

	<-streamArenaStopped
}
