package handler

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/vizserver/types"
	"github.com/gorilla/mux"
)

func Game(fetchVizGames func() ([]*types.VizGame, error), basepath string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)

		vizgames, err := fetchVizGames()
		if err != nil {
			w.Write([]byte("ERROR: Could not fetch viz games"))
			return
		}

		var gameDescription commontypes.GameDescriptionInterface
		foundgame := false

		for _, vizgame := range vizgames {
			if vizgame.GetGame().GetId() == vars["id"] {
				gameDescription = vizgame.GetGame()
				foundgame = true
				break
			}
		}

		if !foundgame {
			w.Write([]byte("GAME NOT FOUND !"))
			return
		}

		vizhtml, err := ioutil.ReadFile(basepath + "index.html")
		if err != nil {
			w.Write([]byte("ERROR: could not render game"))
			return
		}

		protocol := "ws"

		if os.Getenv("ENV") == "prod" {
			protocol = "wss"
		}

		var vizhtmlTemplate = template.Must(template.New("").Parse(string(vizhtml)))
		vizhtmlTemplate.Execute(w, struct {
			WsURL      string
			CDNBaseURL string
			Rand       int64
			Tps        int
		}{
			WsURL: protocol + "://" + r.Host + "/arena/" + gameDescription.GetId() + "/ws",
			Rand:  time.Now().Unix(),
			Tps:   gameDescription.GetTps(),
		})
	}
}
