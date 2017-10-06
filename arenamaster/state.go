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

	bootingVM map[int]Container
	runningVM map[int]Container
}

func NewState() *State {
	return &State{
		idleArenas:    make(map[string]ArenaServerState),
		runningArenas: make(map[string]ArenaServerState),
		pendingArenas: make(map[string]ArenaServerState),

		bootingVM: make(map[int]Container),
		runningVM: make(map[int]Container),
	}
}

func (s *State) UpdateStateAddBootingVM(id int) (stateUpdated bool) {
	s.LockState()

	s.bootingVM[id] = nil

	s.UnlockState()

	stateUpdated = true

	return stateUpdated
}

func (s *State) UpdateStateVMHalted(id int) (stateUpdated bool) {
	s.LockState()

	if _, ok := s.runningVM[id]; ok {
		delete(s.runningVM, id)
		stateUpdated = true
	}

	s.UnlockState()

	return stateUpdated
}

func (s *State) UpdateStateVMBooted(id int, data interface{}) (stateUpdated bool) {
	stateUpdated = false

	s.LockState()

	if _, ok := s.bootingVM[id]; ok {
		delete(s.bootingVM, id)

		s.runningVM[id] = data

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
