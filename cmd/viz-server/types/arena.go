package types

type VizArena struct {
	id   string
	name string
	tps  int
	pool *WatcherMap
}

func NewVizArena(id string, name string, tps int) *VizArena {
	return &VizArena{
		id:   id,
		name: name,
		tps:  tps,
		pool: NewWatcherMap(),
	}
}

func (arena *VizArena) GetId() string {
	return arena.id
}

func (arena *VizArena) GetName() string {
	return arena.name
}

func (arena *VizArena) GetTps() int {
	return arena.tps
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
