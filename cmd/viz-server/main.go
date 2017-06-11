package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	apphandler "github.com/bytearena/bytearena/cmd/viz-server/handler"
	"github.com/bytearena/bytearena/cmd/viz-server/types"
	"github.com/bytearena/bytearena/common/api"
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/messagebroker"
	"github.com/bytearena/bytearena/utils"
)

func main() {

	// => Serveur HTTP
	//		=> Service des assets statiques de la viz (js, modèles, textures)
	// 		=> Ecoute des messages du messagebroker sur le canal viz
	// 		=> Redistribution des messages via websocket
	// 			=> gestion d'un pool de connexions websocket

	webclientpath := utils.GetExecutableDir() + "/webclient/"

	log.Println("Byte Arena Viz Server v0.1; serving assets from " + webclientpath)

	port := flag.Int("port", 8081, "Port of the viz server")
	mqhost := flag.String("mqhost", "mq:5678", "Message queue host:port")
	apiurl := flag.String("apiurl", "http://bytearena.com/privateapi/graphql", "GQL API URL")

	flag.Parse()

	// Connect to Message broker
	mqclient, err := messagebroker.NewClient(*mqhost)
	utils.Check(err, "ERROR: could not connect to messagebroker")

	mqclient.Subscribe("viz", "message", func(msg messagebroker.BrokerMessage) {
		log.Println("RECEIVED viz:message from MESSAGEBROKER; goroutines: " + strconv.Itoa(runtime.NumGoroutine()))
		notify.PostTimeout("viz:message", string(msg.Data), time.Millisecond)
	})

	// Make GraphQL client
	graphqlclient := graphql.MakeClient(*apiurl)

	// Fetch arena instances
	arenainstances, err := api.FetchArenaInstances(graphqlclient)
	utils.Check(err, "Could not fetch arena instances from GraphQL server")

	vizarenas := types.NewVizArenaMap()
	for _, arenainstance := range arenainstances {
		vizarenas.Set(
			arenainstance.GetId(),
			types.NewVizArena(arenainstance),
		)
	}

	logger := os.Stdout

	router := mux.NewRouter()
	router.Handle("/", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Home(vizarenas)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Arena(vizarenas, webclientpath)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Websocket(vizarenas)),
	)).Methods("GET")

	// Les assets de la viz (js, modèles, textures)
	router.PathPrefix("/lib/").Handler(http.FileServer(http.Dir(webclientpath)))
	router.PathPrefix("/res/").Handler(http.FileServer(http.Dir(webclientpath)))

	serverAddr := ":" + strconv.Itoa(*port)
	log.Println("VIZ-SERVER listening on " + serverAddr)

	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Panicln("VIZ-SERVER cannot listen on requested port")
	}
}
