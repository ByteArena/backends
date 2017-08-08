package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/graphql"
	apiqueries "github.com/bytearena/bytearena/common/graphql/queries"
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/vizserver"
)

// Simplified version of the VizMessage struct
type ArenaIdVizMessage struct {
	ArenaId string
	UUID    string
}

func main() {
	env := os.Getenv("ENV")

	// => Serveur HTTP
	//		=> Service des assets statiques de la viz (js, modèles, textures)
	// 		=> Ecoute des messages du messagebroker sur le canal viz
	// 		=> Redistribution des messages via websocket
	// 			=> gestion d'un pool de connexions websocket

	webclientpath := utils.GetExecutableDir() + "/webclient/"

	log.Println("Byte Arena Viz Server v0.1; serving assets from " + webclientpath)

	port := flag.Int("port", 8081, "Port of the viz server")
	mqhost := flag.String("mqhost", "mq:5678", "Message queue host:port")
	apiurl := flag.String("apiurl", "http://graphql.net.bytearena.com", "GQL API URL")
	recordDirectory := flag.String("record-dir", "", "Record files destination")

	flag.Parse()

	// Connect to Message broker
	mqclient, err := mq.NewClient(*mqhost)
	utils.Check(err, "ERROR: could not connect to messagebroker")

	var recorder recording.Recorder = recording.MakeEmptyRecorder()
	if *recordDirectory != "" {
		recorder = recording.MakeMultiArenaRecorder(*recordDirectory)
	}

	mqclient.Subscribe("viz", "message", func(msg mq.BrokerMessage) {
		var vizMessage []ArenaIdVizMessage
		err := json.Unmarshal([]byte(msg.Data), &vizMessage)

		utils.CheckWithFunc(err, func() string {
			return "Failed to decode vizmessage: " + err.Error()
		})

		arenaId := vizMessage[0].ArenaId
		UUID := vizMessage[0].UUID

		recorder.Record(UUID, string(msg.Data))

		utils.Debug("viz:message", "received batch of "+strconv.Itoa(len(vizMessage))+" message(s) for arena "+arenaId)
		notify.PostTimeout("viz:message:"+arenaId, string(msg.Data), time.Millisecond)
	})

	// Make GraphQL client
	graphqlclient := graphql.MakeClient(*apiurl)
	serverAddr := ":" + strconv.Itoa(*port)

	vizservice := vizserver.NewVizService(serverAddr, webclientpath, func() ([]arenaserver.Game, error) {
		games, err := apiqueries.FetchGames(graphqlclient)
		if err != nil {
			return nil, errors.New("Could not fetch games from GraphQL server")
		}

		return games, nil
	}, recorder)

	vizservice.Start()

	var hc *healthcheck.HealthCheckServer
	if env == "prod" {
		hc = NewHealthCheck(mqclient, graphqlclient, "http://"+serverAddr)
		hc.Start()
	}

	<-common.SignalHandler()
	utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")
	vizservice.Stop()

	if hc != nil {
		hc.Stop()
	}
}
