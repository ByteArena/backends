package state

import (
	"testing"
)

func TestQueryState(t *testing.T) {
	data := new(struct{})
	state := NewState()
	id := 1

	state.UpdateStateAddBootingVM(id)
	updated := state.UpdateStateVMBooted(id, data)

	if updated == false {
		panic("State should have been updated")
	}

	queryRes := state.QueryState(id, STATE_RUNNING_VM)

	if queryRes == nil {
		panic("Query should have returned our data")
	}
}

func TestQueryStateNotErrored(t *testing.T) {
	data := new(struct{})
	state := NewState()
	id := 1

	state.UpdateStateAddBootingVM(id)

	updated := state.UpdateStateVMBooted(id, data)
	updated2 := state.UpdateStateVMErrored(id)

	if updated == false || updated2 == false {
		panic("State should have been updated")
	}

	queryRes := state.QueryState(id, STATE_RUNNING_VM)

	if queryRes != nil {
		panic("Should not return data")
	}
}