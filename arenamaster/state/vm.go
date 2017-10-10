package state

func (s *State) UpdateStateAddBootingVM(id int) (stateUpdated bool) {
	s.lockState()

	s.state[id] = &DataContainer{
		Data:   nil,
		Status: STATE_BOOTING_VM,
	}

	s.unlockState()

	stateUpdated = true

	return stateUpdated
}

func (s *State) UpdateStateVMErrored(id int) (stateUpdated bool) {
	s.lockState()

	if state, ok := s.state[id]; ok {
		state.Status |= STATE_ERRORED_VM
		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}

func (s *State) UpdateStateVMHalted(id int) (stateUpdated bool) {
	s.lockState()

	if state, ok := s.state[id]; ok {

		state.Status ^= STATE_IDLE_ARENA
		state.Status ^= STATE_RUNNING_ARENA
		state.Status ^= STATE_PENDING_ARENA

		state.Status ^= STATE_RUNNING_VM
		state.Status |= STATE_HALTED_VM

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}

func (s *State) UpdateStateVMBooted(id int, data interface{}) (stateUpdated bool) {
	stateUpdated = false

	s.lockState()

	if state, ok := s.state[id]; ok {
		state.Status ^= STATE_BOOTING_VM
		state.Status |= STATE_RUNNING_VM

		state.Data = data

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}
