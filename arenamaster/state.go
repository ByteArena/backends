package arenamaster

type ArenaState struct {
	id string
}

type State struct {
	idleArenas    map[string]ArenaState
	runningArenas map[string]ArenaState
}

func NewState() *State {
	return &State{
		idleArenas:    make(map[string]ArenaState),
		runningArenas: make(map[string]ArenaState),
	}
}
