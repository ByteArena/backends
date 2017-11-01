package vizserver

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/bytearena/bytearena/common/mappack"
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
	mapkey        string
	fetchGames    FetchArenasCbk
	listener      *http.Server
	recordStore   recording.RecordStoreInterface
	mappack       *mappack.MappackInMemoryArchive
}

func NewVizService(addr string, webclientpath string, mapkey string, fetchArenas FetchArenasCbk, recordStore recording.RecordStoreInterface, mappack *mappack.MappackInMemoryArchive) *VizService {
	return &VizService{
		addr:          addr,
		webclientpath: webclientpath,
		mapkey:        mapkey,
		fetchGames:    fetchArenas,
		recordStore:   recordStore,
		mappack:       mappack,
	}
}

type GZIPMiddleware struct {
	handler http.Handler
}

func (f GZIPMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "model.json") {
		r.URL.Path += ".gz"
		w.Header().Set("Content-Encoding", "gzip")
	}

	f.handler.ServeHTTP(w, r)
}

type MapRouterMiddleware struct {
	mapkey  string
	handler http.Handler
}

func (m MapRouterMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = "/map/" + m.mapkey + "/" + strings.TrimPrefix(r.URL.Path, "/map/")
	log.Println(r.URL.Path)
	log.Println("MapRouterMiddleware", r.URL.Path)
	m.handler.ServeHTTP(w, r)
}

func (viz *VizService) Start() chan struct{} {

	logger := os.Stdout
	router := mux.NewRouter()

	// Les assets de la viz (js, modèles, textures)
	router.PathPrefix("/lib/").Handler(http.FileServer(http.Dir(viz.webclientpath)))
	router.PathPrefix("/map/").Handler(MapRouterMiddleware{
		mapkey: viz.mapkey,
		handler: GZIPMiddleware{
			handler: http.FileServer(http.Dir(viz.webclientpath)),
		},
	})

	router.Handle("/", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Home(viz.fetchGames)),
	)).Methods("GET")

	router.Handle("/record/{recordId:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Replay(viz.recordStore, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/record/{recordId:[a-zA-Z0-9\\-]+}/ws", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.ReplayWebsocket(viz.recordStore, viz.webclientpath)),
	)).Methods("GET")

	router.Handle("/arena/{id:[a-zA-Z0-9\\-]+}", handlers.CombinedLoggingHandler(logger,
		http.HandlerFunc(apphandler.Game(viz.fetchGames, viz.webclientpath)),
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
	viz.mappack.Close()
	viz.listener.Shutdown(context.TODO())
}
