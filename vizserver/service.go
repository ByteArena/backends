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

type FetchArenasCbk func() ([]arenaserver.ArenaInstance, error)

type VizService struct {
	addr          string
	webclientpath string
	fetchArenas   FetchArenasCbk
	listener      *http.Server
	recorder      recording.Recorder
}

func NewVizService(addr string, webclientpath string, fetchArenas FetchArenasCbk, recorder recording.Recorder) *VizService {
	return &VizService{
		addr:          addr,
		webclientpath: webclientpath,
		fetchArenas:   fetchArenas,
		recorder:      recorder,
	}
}

func (viz *VizService) Start() chan struct{} {

	arenainstances, err := viz.fetchArenas()
	utils.Check(err, "VizService: Could not fetch arenas")

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

	router.Handle("/replay/{recordId:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Replay(viz.recorder, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/replay/{recordId:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.ReplayWebsocket(viz.recorder, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Arena(vizarenas, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Websocket(vizarenas, viz.recorder)),
	)).Methods("GET")

	// Les assets de la viz (js, modèles, textures)
	router.PathPrefix("/lib/").Handler(http.FileServer(http.Dir(viz.webclientpath)))
	router.PathPrefix("/res/").Handler(http.FileServer(http.Dir(viz.webclientpath)))

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
