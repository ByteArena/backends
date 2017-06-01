package types

type Arena struct {
	id   string
	name string
	pool *WatcherMap
}

func NewArena(id string, name string) *Arena {
	return &Arena{
		id:   id,
		name: name,
		pool: NewWatcherMap(),
	}
}

func (arena *Arena) GetId() string {
	return arena.id
}

func (arena *Arena) GetName() string {
	return arena.name
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
