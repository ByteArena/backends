package arenamaster

import (
	"sync"
)

type ArenaServerState struct {
	id     string
	GameId string
}

type State struct {
	mutex sync.Mutex

	idleArenas    map[string]ArenaServerState
	runningArenas map[string]ArenaServerState
	pendingArenas map[string]ArenaServerState
}

func NewState() *State {
	return &State{
		idleArenas:    make(map[string]ArenaServerState),
		runningArenas: make(map[string]ArenaServerState),
		pendingArenas: make(map[string]ArenaServerState),
	}
}

func (s *State) LockState() {
	s.mutex.Lock()
}

func (s *State) UnlockState() {
	s.mutex.Unlock()
}
