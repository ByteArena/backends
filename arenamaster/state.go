package arenamaster

type ArenaState struct {
	id     string
	GameId string
}

type State struct {
	idleArenas    map[string]ArenaState
	runningArenas map[string]ArenaState
	pendingArenas map[string]ArenaState
}

func NewState() *State {
	return &State{
		idleArenas:    make(map[string]ArenaState),
		runningArenas: make(map[string]ArenaState),
		pendingArenas: make(map[string]ArenaState),
	}
}
