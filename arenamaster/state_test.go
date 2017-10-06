package arenamaster

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

	queryRes := state.QueryState(id, STATE_RUNNINGVM)

	if queryRes == nil {
		panic("Query should have returned our data")
	}
}
