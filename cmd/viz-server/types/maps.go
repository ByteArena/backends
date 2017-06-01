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

type ArenaMap struct {
	*commontypes.SyncMap
}

func NewArenaMap() *ArenaMap {
	return &ArenaMap{
		commontypes.NewSyncMap(),
	}
}

func (amap *ArenaMap) Get(id string) *Arena {
	if res, ok := (amap.GetGeneric(id)).(*Arena); ok {
		return res
	}

	return nil
}
