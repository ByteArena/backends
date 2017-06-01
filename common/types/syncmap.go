package types

import "sync"

type SyncMap struct {
	data map[string]interface{}
	lock *sync.RWMutex
}

func NewSyncMap() *SyncMap {
	return &SyncMap{
		data: make(map[string]interface{}, 0),
		lock: &sync.RWMutex{},
	}
}

func (wmap *SyncMap) GetGeneric(id string) interface{} {
	var res interface{}
	present := false

	wmap.lock.RLock()
	if res, present = wmap.data[id]; !present {
		res = nil
	}
	wmap.lock.RUnlock()

	return res
}

func (wmap *SyncMap) Set(id string, item interface{}) error {
	wmap.lock.Lock()
	wmap.data[id] = item
	wmap.lock.Unlock()

	return nil
}

func (wmap *SyncMap) Remove(id string) {
	wmap.lock.Lock()
	delete(wmap.data, id)
	wmap.lock.Unlock()
}

func (wmap *SyncMap) Size() int {
	return len(wmap.data)
}
