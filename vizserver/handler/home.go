package handler

import (
	"net/http"
	"strconv"

	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/vizserver/types"
)

func Home(fetchVizGames func() ([]*types.VizGame, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<h2>Welcome on VIZ SERVER !</h2>"))

		vizgames, err := fetchVizGames()
		if err != nil {
			utils.Debug("viz-server", "Home handler(): could not fetch games; "+err.Error())
			return
		}

		for _, vizgame := range vizgames {
			game := vizgame.GetGame()
			gameStatusLabel := "pending"
			switch game.GetRunStatus() {
			case gqltypes.GameRunStatus.Pending:
				{
					gameStatusLabel = "Pending"
				}
			case gqltypes.GameRunStatus.Running:
				{
					gameStatusLabel = "Running"
				}
			case gqltypes.GameRunStatus.Finished:
				{
					gameStatusLabel = "Finished"
				}
			}

			w.Write([]byte("<a href='/arena/" + game.GetId() + "'>" + game.GetName() + " (" + strconv.Itoa(vizgame.GetNumberWatchers()) + " watchers right now) &mdash; " + gameStatusLabel + "</a><br />"))
		}
	}
}
