package arenamaster

type ArenaState struct {
	id string
}

type State struct {
	arenas []ArenaState
}

func NewState() *State {
	return &State{
		arenas: make([]ArenaState, 0),
	}
}
