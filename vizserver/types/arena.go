package types

import (
	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type VizGame struct {
	game arenaserver.GameInterface
	pool *WatcherMap
}

func NewVizGame(game arenaserver.GameInterface) *VizGame {
	return &VizGame{
		pool: NewWatcherMap(),
		game: game,
	}
}

func (vizgame *VizGame) GetGame() arenaserver.GameInterface {
	return vizgame.game
}

func (vizgame *VizGame) SetGame(game arenaserver.GameInterface) {
	vizgame.game = game
}

func (vizgame *VizGame) GetTps() int {
	return vizgame.game.GetTps()
}

type VizInitMessageData struct {
	Map *mapcontainer.MapContainer `json:"map"`
}

type VizInitMessage struct {
	Type string             `json:"type"`
	Data VizInitMessageData `json:"data"`
}

func (vizgame *VizGame) SetWatcher(watcher *Watcher) {
	vizgame.pool.Set(watcher.GetId(), watcher)

	initMsg := VizInitMessage{
		Type: "init",
		Data: VizInitMessageData{
			Map: vizgame.game.GetMapContainer(),
		},
	}

	err := watcher.conn.WriteJSON(initMsg)
	utils.Check(err, "Could not send VizInitMessage JSON")
}

func (vizgame *VizGame) RemoveWatcher(watcherid string) {
	vizgame.pool.Remove(watcherid)
}

func (vizgame *VizGame) GetNumberWatchers() int {
	return vizgame.pool.Size()
}
