package vizserver

import (
	"net"
	"net/http"
	"os"

	"log"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	apphandler "github.com/bytearena/bytearena/vizserver/handler"
	"github.com/bytearena/bytearena/vizserver/types"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type FetchArenasCbk func() ([]arenaserver.Game, error)

type VizService struct {
	addr          string
	webclientpath string
	fetchArenas   FetchArenasCbk
	listener      *http.Server
	recorder      recording.Recorder
	pathToAssets  string
}

func NewVizService(addr string, webclientpath string, fetchArenas FetchArenasCbk, recorder recording.Recorder) *VizService {
	return &VizService{
		addr:          addr,
		webclientpath: webclientpath,
		fetchArenas:   fetchArenas,
		recorder:      recorder,
	}
}

func (viz *VizService) SetPathToAssets(path string) {
	viz.pathToAssets = path
}

func (viz *VizService) Start() chan struct{} {

	games, err := viz.fetchArenas()
	utils.Check(err, "VizService: Could not fetch arenas")

	vizarenas := types.NewVizArenaMap()
	for _, game := range games {
		vizarenas.Set(
			game.GetId(),
			types.NewVizArena(game),
		)
	}

	logger := os.Stdout
	router := mux.NewRouter()

	// Les assets de la viz (js, modèles, textures)
	router.PathPrefix("/lib/").Handler(http.StripPrefix("/lib/", http.FileServer(http.Dir(viz.webclientpath+"/lib/"))))
	cdnBaseURL := "https://bytearena.com/assets/bytearena"

	if viz.pathToAssets != "" {
		router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir(viz.pathToAssets))))
		cdnBaseURL = "http://localhost:8081/assets"
	}

	router.Handle("/", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Home(vizarenas)),
	)).Methods("GET")

	router.Handle("/record/{recordId:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Replay(viz.recorder, viz.webclientpath, cdnBaseURL)),
	)).Methods("GET")

	router.Handle("/record/{recordId:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.ReplayWebsocket(viz.recorder, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Arena(vizarenas, viz.webclientpath, cdnBaseURL)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Websocket(vizarenas, viz.recorder)),
	)).Methods("GET")

	log.Println("VIZ Listening on " + viz.addr)

	listener, err := net.Listen("tcp4", viz.addr)
	if err != nil {
		utils.Check(err, err.Error())
	}

	viz.listener = &http.Server{
		Handler: router,
	}

	block := make(chan struct{})

	go func(block chan struct{}) {
		err := viz.listener.Serve(listener)
		utils.Check(err, "Failed to listen on "+viz.addr)
		close(block)
	}(block)

	return block
}

func (viz *VizService) Stop() {
	viz.listener.Shutdown(nil)
}
