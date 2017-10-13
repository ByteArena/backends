package state

import (
	"testing"
)

type testCase struct {
	Name         string
	InitialState byte
	ResultState  byte
	Mutations    func(s *State, id int)
}

func TestQueryState(t *testing.T) {
	data := new(struct{})
	state := NewState()
	id := 1

	state.UpdateStateAddBootingVM(id, data)

	queryRes := state.QueryState(id, STATE_BOOTING_VM)

	if queryRes == nil {
		panic("Query should have returned our data")
	}
}

func TestQueryStateNotErrored(t *testing.T) {
	data := new(struct{})
	state := NewState()
	id := 1

	state.UpdateStateAddBootingVM(id, data)

	updated := state.UpdateStateVMBooted(id)
	updated2 := state.UpdateStateVMErrored(id)

	if updated == false || updated2 == false {
		panic("State should have been updated")
	}

	queryRes := state.QueryState(id, STATE_RUNNING_VM)

	if queryRes != nil {
		panic("Should not return data")
	}
}
