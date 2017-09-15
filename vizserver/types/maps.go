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

type VizGameMap struct {
	*commontypes.SyncMap
}

func NewVizGameMap() *VizGameMap {
	return &VizGameMap{
		commontypes.NewSyncMap(),
	}
}

func (amap *VizGameMap) Get(id string) *VizGame {
	if res, ok := (amap.GetGeneric(id)).(*VizGame); ok {
		return res
	}

	return nil
}
