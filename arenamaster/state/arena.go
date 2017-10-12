package state

func (s *State) UpdateStateAddIdleArena(id int) (stateUpdated bool) {
	stateUpdated = false

	s.lockState()

	if state, ok := s.state[id]; ok {
		if state.Status&STATE_BOOTING_VM != 0 {
			state.Status ^= STATE_BOOTING_VM
		}

		state.Status |= STATE_IDLE_ARENA

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}

func (s *State) UpdateStateTriedLaunchArena(id int) (stateUpdated bool) {
	stateUpdated = false

	s.lockState()

	if state, ok := s.state[id]; ok {
		state.Status ^= STATE_IDLE_ARENA
		state.Status |= STATE_PENDING_ARENA

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}

func (s *State) UpdateStateConfirmedLaunchArena(id int) (stateUpdated bool) {
	stateUpdated = false

	s.lockState()

	if state, ok := s.state[id]; ok {
		state.Status ^= STATE_PENDING_ARENA
		state.Status |= STATE_RUNNING_ARENA

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}

func (s *State) UpdateStateStoppedArena(id int) (stateUpdated bool) {
	stateUpdated = false

	s.lockState()

	if state, ok := s.state[id]; ok {
		if state.Status&STATE_IDLE_ARENA != 0 {
			state.Status ^= STATE_IDLE_ARENA
		}

		state.Status ^= STATE_RUNNING_ARENA

		stateUpdated = true
	}

	s.unlockState()

	return stateUpdated
}
