package arenamaster

type ArenaState struct {
	uuid string
}

type State struct {
	arenas []ArenaState
}

func NewState() *State {
	return &State{
		arenas: make([]ArenaState, 0),
	}
}
