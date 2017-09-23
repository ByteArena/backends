package types

import (
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type VizGame struct {
	gameDescription types.GameDescriptionInterface
	pool            *WatcherMap
}

func NewVizGame(gameDescription types.GameDescriptionInterface) *VizGame {
	return &VizGame{
		pool:            NewWatcherMap(),
		gameDescription: gameDescription,
	}
}

func (vizgame *VizGame) GetGame() types.GameDescriptionInterface {
	return vizgame.gameDescription
}

func (vizgame *VizGame) SetGame(game types.GameDescriptionInterface) {
	vizgame.gameDescription = game
}

func (vizgame *VizGame) GetTps() int {
	return vizgame.gameDescription.GetTps()
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
			Map: vizgame.gameDescription.GetMapContainer(),
		},
	}

	err := watcher.conn.WriteJSON(initMsg)
	if err != nil {
		utils.Debug("viz-server", "Could not send VizInitMessage JSON;"+err.Error())
	}
}

func (vizgame *VizGame) RemoveWatcher(watcherid string) {
	vizgame.pool.Remove(watcherid)
}

func (vizgame *VizGame) GetNumberWatchers() int {
	return vizgame.pool.Size()
}
