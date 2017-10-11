package state

import (
	"sync"
)

const (
	STATE_BOOTING_VM byte = 1 << iota
	STATE_RUNNING_VM
	STATE_HALTED_VM
	STATE_ERRORED_VM
	STATE_IDLE_ARENA
	STATE_RUNNING_ARENA
	STATE_PENDING_ARENA
	STATE_ERRORED_ARENA
)

type Data interface{}
type DataContainer struct {
	Data   Data
	Status byte
}

type State struct {
	mutex sync.Mutex

	state map[int]*DataContainer
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

func (s *State) GetStatus(id int) byte {
	if state, hasInState := s.state[id]; hasInState {
		return state.Status
	} else {
		return 0
	}
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
		state: make(map[int]*DataContainer),
	}
}

func (s *State) QueryState(id int, flag byte) Data {
	if data, ok := s.state[id]; ok {
		if data.Status&flag != 0 && data.Status&STATE_ERRORED_ARENA == 0 && data.Status&STATE_ERRORED_VM == 0 {
			return data.Data
		} else {
			return nil
		}
	}

	return nil
}

func (s *State) FindState(flag byte) Data {
	s.lockState()

	for _, data := range s.state {

		if data.Status&flag != 0 {
			s.unlockState()
			return data.Data
		}
	}

	s.unlockState()
	return nil
}

func (s *State) lockState() {
	s.mutex.Lock()
}

func (s *State) unlockState() {
	s.mutex.Unlock()
}

func (s *State) Map(fn func(element *DataContainer)) {
	s.lockState()

	for _, element := range s.state {
		fn(element)
	}

	s.unlockState()
}

func (s *State) remove(id int) {
	delete(s.state, id)
}
