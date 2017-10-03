package arenamaster

import (
	"sync"
)

type ArenaServerState struct {
	id     string
	GameId string
}

type Container interface{}

type State struct {
	mutex sync.Mutex

	idleArenas    map[string]ArenaServerState
	runningArenas map[string]ArenaServerState
	pendingArenas map[string]ArenaServerState

	bootingVM map[string]Container
	runningVM map[string]Container
}

func NewState() *State {
	return &State{
		idleArenas:    make(map[string]ArenaServerState),
		runningArenas: make(map[string]ArenaServerState),
		pendingArenas: make(map[string]ArenaServerState),

		bootingVM: make(map[string]Container),
		runningVM: make(map[string]Container),
	}
}

func (s *State) UpdateAddBootingVM(name string) (stateUpdated bool) {
	s.LockState()

	s.bootingVM[name] = nil

	s.UnlockState()

	stateUpdated = true

	return stateUpdated
}

func (s *State) UpdateVMHalted(name string) (stateUpdated bool) {
	s.LockState()

	if _, ok := s.runningVM[name]; ok {
		delete(s.runningVM, name)
		stateUpdated = true
	}

	s.UnlockState()

	return stateUpdated
}

func (s *State) UpdateVMBooted(name string, data interface{}) (stateUpdated bool) {
	stateUpdated = false

	s.LockState()

	if _, ok := s.bootingVM[name]; ok {
		delete(s.bootingVM, name)

		s.runningVM[name] = data

		stateUpdated = true
	}

	s.UnlockState()

	return stateUpdated
}

// TODO(sven): don't expose
func (s *State) LockState() {
	s.mutex.Lock()
}

// TODO(sven): don't expose
func (s *State) UnlockState() {
	s.mutex.Unlock()
}
