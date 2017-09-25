package arenamaster

import (
	"sync"

	"github.com/bytearena/bytearena/common/utils"
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
	utils.Debug("state", "locking")
	s.mutex.Lock()
}

func (s *State) UnlockState() {
	utils.Debug("state", "unlocking")
	s.mutex.Unlock()
}
