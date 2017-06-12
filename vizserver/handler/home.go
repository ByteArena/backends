package handler

import (
	"net/http"
	"strconv"

	"github.com/bytearena/bytearena/vizserver/types"
)

func Home(arenas *types.VizArenaMap) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<h2>Welcome on VIZ SERVER !</h2>"))

		arenasArray := arenas.ToArrayGeneric()

		for _, item := range arenasArray {
			if arena, ok := item.(*types.VizArena); ok {
				w.Write([]byte("<a href='/arena/" + arena.GetId() + "'>" + arena.GetName() + " (" + strconv.Itoa(arena.GetNumberWatchers()) + " watchers right now)</a><br />"))
			}
		}
	}
}
