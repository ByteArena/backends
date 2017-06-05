package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bytearena/bytearena/cmd/viz-server/types"
	"github.com/gorilla/mux"
)

func arenaHandler(arenas *types.VizArenaMap, basepath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		arena := arenas.Get(vars["id"])

		if arena == nil {
			w.Write([]byte("ARENA NOT FOUND !"))
			return
		}

		vizhtml, err := ioutil.ReadFile(basepath + "index.html")
		if err != nil {
			w.Write([]byte("ERROR: could not render arena"))
			return
		}

		var vizhtmlTemplate = template.Must(template.New("").Parse(string(vizhtml)))
		vizhtmlTemplate.Execute(w, struct {
			WsURL string
			Rand  int64
			Tps   int
		}{
			WsURL: "ws://" + r.Host + "/arena/" + arena.GetId() + "/ws",
			Rand:  time.Now().Unix(),
			Tps:   arena.GetTps(),
		})
	}
}
