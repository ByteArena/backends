package handler

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/bytearena/bytearena/arenaserver"

	"github.com/bytearena/bytearena/vizserver/types"
	"github.com/gorilla/mux"
)

func Game(fetchVizGames func() ([]*types.VizGame, error), basepath string, CDNBaseURL string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)

		vizgames, err := fetchVizGames()
		if err != nil {
			w.Write([]byte("ERROR: Could not fetch viz games"))
			return
		}

		var game arenaserver.GameInterface
		foundgame := false

		for _, vizgame := range vizgames {
			if vizgame.GetGame().GetId() == vars["id"] {
				game = vizgame.GetGame()
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
			WsURL:      protocol + "://" + r.Host + "/arena/" + game.GetId() + "/ws",
			CDNBaseURL: CDNBaseURL,
			Rand:       time.Now().Unix(),
			Tps:        game.GetTps(),
		})
	}
}
