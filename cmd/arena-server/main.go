package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/graphql"
	apiqueries "github.com/bytearena/bytearena/common/graphql/queries"
	"github.com/bytearena/bytearena/common/healthcheck"

	"github.com/bytearena/bytearena/arenaserver/container"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/protocol"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

type messageArenaLaunch struct {
	Id string `json:"id"`
}

func main() {
	env := os.Getenv("ENV")

	rand.Seed(time.Now().UnixNano())

	host := flag.String("host", "", "IP serving the arena; required")
	arenaServerUUID := flag.String("id", "", "ID of the arena; required")
	port := flag.Int("port", 8080, "Port serving the arena")
	mqhost := flag.String("mqhost", "mq:5678", "Message queue host:port")
	apiurl := flag.String("apiurl", "https://graphql.net.bytearena.com", "GQL API URL")
	timeout := flag.Int("timeout", 60, "Limit the time of the game (in minutes)")
	registryAddr := flag.String("registryAddr", "", "Docker registry address")
	arenaAddr := flag.String("arenaAddr", "", "Address of the arena")

	flag.Parse()

	utils.Assert((*arenaServerUUID) != "", "id must be set")
	utils.Assert((*registryAddr) != "", "Docker registry address must be set")
	utils.Assert((*arenaAddr) != "", "Arena address must be set")

	log.Println("Byte Arena Server v0.1 ID#" + (*arenaServerUUID))

	// Make GraphQL client
	graphqlclient := graphql.MakeClient(*apiurl)

	// Make message broker client
	brokerclient, err := mq.NewClient(*mqhost)
	utils.Check(err, "ERROR: Could not connect to messagebroker on "+*mqhost)

	brokerclient.Publish("game", "handshake", types.NewMQMessage(
		"arena-server",
		"Arena Server "+(*arenaServerUUID)+" reporting for duty.",
	).SetPayload(types.MQPayload{
		"id": (*arenaServerUUID),
	}))

	streamArenaLaunched := make(chan interface{})
	notify.Start("game:launch", streamArenaLaunched)

	brokerclient.Subscribe("game", (*arenaServerUUID)+".launch", func(msg mq.BrokerMessage) {
		utils.Debug("arenamaster", "Received launching order")

		var payload messageArenaLaunch
		err := json.Unmarshal(msg.Data, &payload)
		if err != nil {
			log.Println(err)
			log.Println("ERROR:game:launch Invalid payload " + string(msg.Data))
			return
		}

		notify.PostTimeout("game:launch", payload, time.Millisecond)
	})

	var hc *healthcheck.HealthCheckServer
	if env == "prod" {
		hc = NewHealthCheck(brokerclient, graphqlclient)
		hc.Start()
	}

	go func() {
		for {
			select {
			case payload := <-streamArenaLaunched:
				{
					if arenaSubmitted, ok := payload.(messageArenaLaunch); ok {

						// Fetch game from GraphQL
						arena, err := apiqueries.FetchGameById(graphqlclient, arenaSubmitted.Id)
						utils.Check(err, "Could not fetch game "+arenaSubmitted.Id)

						orch := container.MakeRemoteContainerOrchestrator(*arenaAddr, *registryAddr)
						srv := arenaserver.NewServer(*host, *port, orch, arena, *arenaServerUUID, brokerclient)

						for _, contestant := range arena.GetContestants() {
							srv.RegisterAgent(contestant.AgentRegistry + "/" + contestant.AgentImage)
						}

						// handling signals
						go func() {
							<-common.SignalHandler()
							utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")
							srv.Stop()
							log.Println("Stop")
						}()

						go protocol.StreamState(srv, brokerclient, *arenaServerUUID)

						// Limit the game in time
						timeoutTimer := time.NewTimer(time.Duration(*timeout) * time.Minute)
						go func() {
							<-timeoutTimer.C

							srv.Stop()
							utils.Debug("timer", "Timeout, stop the arena")
						}()

						serverChan, startErr := srv.Start()

						if startErr != nil {
							srv.Stop()
							log.Panicln("Cannot start server: " + startErr.Error())
						}

						<-serverChan

						notify.PostTimeout("game:stopped", nil, time.Millisecond)
					}
				}
			}
		}
	}()

	streamArenaStopped := make(chan interface{})
	notify.Start("game:stopped", streamArenaStopped)

	<-streamArenaStopped

	if hc != nil {
		log.Println("Stop healthcheck")
		hc.Stop()
	}
}
