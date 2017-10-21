package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVMState(t *testing.T) {
	var data interface{} = nil
	id := 1

	examples := []testCase{
		{
			Name: "Should add a booting VM",

			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateAddBootingVM(id, data))
			},
			ResultState: STATE_BOOTING_VM,
		},
		{
			Name: "Should add a errored VM while booting",

			InitialState: STATE_BOOTING_VM,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateVMErrored(id))
			},
			ResultState: STATE_BOOTING_VM | STATE_ERRORED_VM,
		},
		{
			Name: "Should remove from state a halted VM",

			InitialState: STATE_BOOTING_VM,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateVMHalted(id))
			},
		},
		{
			Name: "Should add a booted VM",

			InitialState: STATE_BOOTING_VM,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateVMBooted(id))
			},
			ResultState: STATE_RUNNING_VM,
		},
	}

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			s := NewState()
			s.create(id, data, example.InitialState)

			example.Mutations(s, id)

			assert.Equal(
				t,
				s.DebugGetStatus(id),
				s.DebugFlagToString(example.ResultState),
			)
		})
	}
}
