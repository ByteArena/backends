package types

import (
	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type VizArena struct {
	game arenaserver.Game
	pool *WatcherMap
}

func NewVizArena(game arenaserver.Game) *VizArena {
	return &VizArena{
		pool: NewWatcherMap(),
		game: game,
	}
}

func (arena *VizArena) GetId() string {
	return arena.game.GetId()
}

func (arena *VizArena) GetName() string {
	return arena.game.GetName()
}

func (arena *VizArena) GetTps() int {
	return arena.game.GetTps()
}

type VizInitMessageData struct {
	Map *mapcontainer.MapContainer `json:"map"`
}

type VizInitMessage struct {
	Type string             `json:"type"`
	Data VizInitMessageData `json:"data"`
}

func (arena *VizArena) SetWatcher(watcher *Watcher) {
	arena.pool.Set(watcher.GetId(), watcher)

	initMsg := VizInitMessage{
		Type: "init",
		Data: VizInitMessageData{
			Map: arena.game.GetMapContainer(),
		},
	}

	err := watcher.conn.WriteJSON(initMsg)
	utils.Check(err, "Could not send VizInitMessage JSON")
}

func (arena *VizArena) RemoveWatcher(watcherid string) {
	arena.pool.Remove(watcherid)
}

func (arena *VizArena) GetNumberWatchers() int {
	return arena.pool.Size()
}
