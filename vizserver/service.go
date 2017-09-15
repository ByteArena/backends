package vizserver

import (
	"net"
	"net/http"
	"os"

	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	apphandler "github.com/bytearena/bytearena/vizserver/handler"
	"github.com/bytearena/bytearena/vizserver/types"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type FetchArenasCbk func() ([]*types.VizGame, error)

type VizService struct {
	addr          string
	webclientpath string
	fetchGames    FetchArenasCbk
	listener      *http.Server
	recordStore   recording.RecordStoreInterface
	pathToAssets  string
}

func NewVizService(addr string, webclientpath string, fetchArenas FetchArenasCbk, recordStore recording.RecordStoreInterface) *VizService {
	return &VizService{
		addr:          addr,
		webclientpath: webclientpath,
		fetchGames:    fetchArenas,
		recordStore:   recordStore,
	}
}

func (viz *VizService) SetPathToAssets(path string) {
	viz.pathToAssets = path
}

func (viz *VizService) Start() chan struct{} {

	logger := os.Stdout
	router := mux.NewRouter()

	// Les assets de la viz (js, modèles, textures)
	router.PathPrefix("/lib/").Handler(http.StripPrefix("/lib/", http.FileServer(http.Dir(viz.webclientpath+"/lib/"))))
	cdnBaseURL := "https://static.bytearena.com/assets/bytearena"

	if viz.pathToAssets != "" {
		router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir(viz.pathToAssets))))
		cdnBaseURL = "/assets"
	}

	router.Handle("/", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Home(viz.fetchGames)),
	)).Methods("GET")

	router.Handle("/record/{recordId:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Replay(viz.recordStore, viz.webclientpath, cdnBaseURL)),
	)).Methods("GET")

	router.Handle("/record/{recordId:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.ReplayWebsocket(viz.recordStore, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Game(viz.fetchGames, viz.webclientpath, cdnBaseURL)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Websocket(viz.fetchGames)),
	)).Methods("GET")

	utils.Debug("viz-server", "VIZ Listening on "+viz.addr)

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
