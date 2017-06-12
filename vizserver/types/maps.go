package types

import (
	commontypes "github.com/bytearena/bytearena/common/types"
)

type WatcherMap struct {
	*commontypes.SyncMap
}

func NewWatcherMap() *WatcherMap {
	return &WatcherMap{
		commontypes.NewSyncMap(),
	}
}

func (wmap *WatcherMap) Get(id string) *Watcher {
	if res, ok := (wmap.GetGeneric(id)).(*Watcher); ok {
		return res
	}

	return nil
}

type VizArenaMap struct {
	*commontypes.SyncMap
}

func NewVizArenaMap() *VizArenaMap {
	return &VizArenaMap{
		commontypes.NewSyncMap(),
	}
}

func (amap *VizArenaMap) Get(id string) *VizArena {
	if res, ok := (amap.GetGeneric(id)).(*VizArena); ok {
		return res
	}

	return nil
}
