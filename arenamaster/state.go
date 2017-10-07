package arenamaster

import (
	"fmt"
	"sync"
)

const (
	STATE_BOOTING_VM    byte = 1 << iota // 00000001
	STATE_RUNNING_VM                     // 00000010
	STATE_HALTED_VM                      // 00000100
	STATE_ERRORED_VM                     // 00001000
	STATE_IDLE_ARENA                     // 00010000
	STATE_RUNNING_ARENA                  // 00100000
	STATE_PENDING_ARENA                  // 01000000
	STATE_ERRORED_ARENA                  // 10000000
)

type ArenaServerState struct {
	id     string
	GameId string
}

type Data interface{}
type DataContainer struct {
	Data   Data
	Status byte
}

type State struct {
	mutex sync.Mutex

	state map[int]*DataContainer

	idleArenas    map[string]ArenaServerState
	runningArenas map[string]ArenaServerState
	pendingArenas map[string]ArenaServerState
}

func (s *State) DebugGetStateDistribution() map[string]int {
	res := make(map[string]int)

	res["STATE_BOOTING_VM"] = 0
	res["STATE_RUNNING_VM"] = 0
	res["STATE_HALTED_VM"] = 0
	res["STATE_ERRORED_VM"] = 0
	res["STATE_IDLE_ARENA"] = 0
	res["STATE_RUNNING_ARENA"] = 0
	res["STATE_PENDING_ARENA"] = 0
	res["STATE_ERRORED_ARENA"] = 0

	for k, _ := range s.state {
		for _, status := range s.DebugGetStatus(k) {
			res[status]++
		}
	}

	return res
}

func (s *State) DebugGetStatus(id int) []string {
	res := make([]string, 0)
	bin := s.state[id].Status

	if bin&STATE_BOOTING_VM != 0 {
		res = append(res, "STATE_BOOTING_VM")
	}

	if bin&STATE_RUNNING_VM != 0 {
		res = append(res, "STATE_RUNNING_VM")
	}

	if bin&STATE_HALTED_VM != 0 {
		res = append(res, "STATE_HALTED_VM")
	}

	if bin&STATE_ERRORED_VM != 0 {
		res = append(res, "STATE_ERRORED_VM")
	}

	if bin&STATE_IDLE_ARENA != 0 {
		res = append(res, "STATE_IDLE_ARENA")
	}

	if bin&STATE_RUNNING_ARENA != 0 {
		res = append(res, "STATE_RUNNING_ARENA")
	}

	if bin&STATE_PENDING_ARENA != 0 {
		res = append(res, "STATE_PENDING_ARENA")
	}

	if bin&STATE_ERRORED_ARENA != 0 {
		res = append(res, "STATE_ERRORED_ARENA")
	}

	return res
}

func NewState() *State {
	return &State{
		idleArenas:    make(map[string]ArenaServerState),
		runningArenas: make(map[string]ArenaServerState),
		pendingArenas: make(map[string]ArenaServerState),

		state: make(map[int]*DataContainer),
	}
}

func (s *State) QueryState(id int, flag byte) Data {
	if data, ok := s.state[id]; ok {
		if data.Status&flag != 0 {
			return data.Data
		} else {
			return nil
		}
	}

	return nil
}

func (s *State) UpdateStateAddBootingVM(id int) (stateUpdated bool) {
	s.LockState()

	s.state[id] = &DataContainer{
		Data:   nil,
		Status: STATE_BOOTING_VM,
	}

	s.UnlockState()

	stateUpdated = true

	return stateUpdated
}

func (s *State) UpdateStateVMErrored(id int) (stateUpdated bool) {
	s.LockState()

	if state, ok := s.state[id]; ok {
		state.Status |= STATE_ERRORED_VM
		stateUpdated = true
	}

	s.UnlockState()

	return stateUpdated
}

func (s *State) UpdateStateVMHalted(id int) (stateUpdated bool) {
	s.LockState()

	if state, ok := s.state[id]; ok {
		state.Status ^= STATE_RUNNING_VM
		state.Status |= STATE_HALTED_VM

		stateUpdated = true
	}

	s.UnlockState()

	return stateUpdated
}

func (s *State) UpdateStateVMBooted(id int, data interface{}) (stateUpdated bool) {
	stateUpdated = false

	s.LockState()

	if state, ok := s.state[id]; ok {
		state.Status ^= STATE_BOOTING_VM
		state.Status |= STATE_RUNNING_VM

		state.Data = data

		fmt.Printf("%b\n", state.Status)

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
