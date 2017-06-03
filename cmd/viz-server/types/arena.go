package types

type Arena struct {
	id   string
	name string
	tps  int
	pool *WatcherMap
}

func NewArena(id string, name string, tps int) *Arena {
	return &Arena{
		id:   id,
		name: name,
		tps:  tps,
		pool: NewWatcherMap(),
	}
}

func (arena *Arena) GetId() string {
	return arena.id
}

func (arena *Arena) GetName() string {
	return arena.name
}

func (arena *Arena) GetTps() int {
	return arena.tps
}

func (arena *Arena) SetWatcher(watcher *Watcher) {
	arena.pool.Set(watcher.GetId(), watcher)
}

func (arena *Arena) RemoveWatcher(watcherid string) {
	arena.pool.Remove(watcherid)
}

func (arena *Arena) GetNumberWatchers() int {
	return arena.pool.Size()
}
