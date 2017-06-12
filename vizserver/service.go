package vizserver

import (
	"net/http"
	"os"

	"log"

	"github.com/bytearena/bytearena/arenaserver"
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
}

func NewVizService(addr string, webclientpath string, fetchArenas FetchArenasCbk) *VizService {
	return &VizService{
		addr:          addr,
		webclientpath: webclientpath,
		fetchArenas:   fetchArenas,
	}
}

func (viz *VizService) ListenAndServe() error {

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

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Arena(vizarenas, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Websocket(vizarenas)),
	)).Methods("GET")

	// Les assets de la viz (js, modèles, textures)
	router.PathPrefix("/lib/").Handler(http.FileServer(http.Dir(viz.webclientpath)))
	router.PathPrefix("/res/").Handler(http.FileServer(http.Dir(viz.webclientpath)))

	log.Println("VIZ Listening on " + viz.addr)

	return http.ListenAndServe(viz.addr, router)
}
