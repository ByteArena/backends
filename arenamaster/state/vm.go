package state

func (s *State) UpdateStateAddBootingVM(id int, data interface{}) (stateUpdated bool) {
	s.lockState()

	s.state[id] = &DataContainer{
		Data:   data,
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
		s.remove(id)

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}

func (s *State) UpdateStateVMBooted(id int) (stateUpdated bool) {
	stateUpdated = false

	s.lockState()

	if state, ok := s.state[id]; ok {
		state.Status ^= STATE_BOOTING_VM
		state.Status |= STATE_RUNNING_VM

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}
