package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArenaState(t *testing.T) {
	var data interface{} = nil
	id := 1

	examples := []testCase{
		{
			Name: "Should add an idle arena",

			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateAddIdleArena(id))
			},
			ResultState: STATE_IDLE_ARENA,
		},
		{
			Name: "Should add an idle arena and remove booting",

			InitialState: STATE_BOOTING_VM,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateAddIdleArena(id))
			},
			ResultState: STATE_IDLE_ARENA,
		},
		{
			Name: "Should add an launched arena",

			InitialState: STATE_IDLE_ARENA,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateTriedLaunchArena(id))
			},
			ResultState: STATE_PENDING_ARENA,
		},
		{
			Name: "Should add an launched and confirmed arena",

			InitialState: STATE_PENDING_ARENA,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateConfirmedLaunchArena(id))
			},
			ResultState: STATE_RUNNING_ARENA,
		},
		{
			Name: "Should add an stopped arena while running",

			InitialState: STATE_RUNNING_ARENA,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateStoppedArena(id))
			},
			ResultState: 0,
		},
		{
			Name: "Should add an stopped arena while idle",

			InitialState: STATE_IDLE_ARENA,
			Mutations: func(s *State, id int) {
				assert.True(t, s.UpdateStateStoppedArena(id))
			},
			ResultState: 0,
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
