package types

import (
	"github.com/bytearena/bytearena/server"
)

type VizArena struct {
	arenainstance server.ArenaInstance
	pool          *WatcherMap
}

func NewVizArena(arenainstance server.ArenaInstance) *VizArena {
	return &VizArena{
		pool:          NewWatcherMap(),
		arenainstance: arenainstance,
	}
}

func (arena *VizArena) GetId() string {
	return arena.arenainstance.GetId()
}

func (arena *VizArena) GetName() string {
	return arena.arenainstance.GetName()
}

func (arena *VizArena) GetTps() int {
	return arena.arenainstance.GetTps()
}

func (arena *VizArena) SetWatcher(watcher *Watcher) {
	arena.pool.Set(watcher.GetId(), watcher)
}

func (arena *VizArena) RemoveWatcher(watcherid string) {
	arena.pool.Remove(watcherid)
}

func (arena *VizArena) GetNumberWatchers() int {
	return arena.pool.Size()
}
