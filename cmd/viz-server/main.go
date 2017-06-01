package main

import (
	"log"
	"net/http"
	"os"

	notify "github.com/bitly/go-notify"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/bytearena/bytearena/cmd/viz-server/types"
	"github.com/bytearena/bytearena/common/messagebroker"
	"github.com/bytearena/bytearena/utils"
)

func main() {

	port, exists := os.LookupEnv("PORT")
	utils.Assert(exists, "Error: PORT should be defined in the environment")

	brokerhost, exists := os.LookupEnv("MESSAGEBROKERHOST")
	utils.Assert(exists, "Error: MESSAGEBROKERHOST should be defined in the environment")

	// Home : liste des arènes en cours de diffusion avec URL et affichage du nombre d'auditeurs
	// /arena/id : visualisation de l'arène

	// => Serveur HTTP
	// => Ecoute des messages du messagebroker sur le canal viz
	// 		=> Initialisation d'une arène : viz:init
	// 		=> Démarrage d'une arène : viz:start
	// 		=> State pour un tick : viz:frame
	// 		=> Arrêt d'une arène : viz:stop
	// => Redistribution des messages via websocket
	// 		=> gestion d'un pool de connexions websocket

	brokerclient, err := messagebroker.NewClient(brokerhost)
	utils.Check(err, "ERROR: could not connect to messagebroker")

	brokerclient.Subscribe("viz", "message", func(msg messagebroker.BrokerMessage) {
		log.Println("RECEIVED viz:message from MESSAGEBROKER")
		notify.Post("viz:message", string(msg.Data))
	})

	arenas := types.NewArenaMap()
	sandboxarena := types.NewArena("sandboxarena", "Sandbox Arena !")
	arenas.Set(sandboxarena.GetId(), sandboxarena)

	logger := os.Stdout

	router := mux.NewRouter()
	router.Handle("/", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(homeHandler()),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(arenaHandler(arenas, "./webclient/")),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(websocketHandler(arenas)),
	)).Methods("GET")

	serverAddr := ":" + port
	log.Println("VIZ-SERVER listening on " + serverAddr)

	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Panicln("VIZ-SERVER cannot listen on requested port")
	}
}
