package arenamaster

type ArenaServerState struct {
	id     string
	GameId string
}

type State struct {
	idleArenas    map[string]ArenaServerState
	runningArenas map[string]ArenaServerState
	pendingArenas map[string]ArenaServerState
}

func NewState() *State {
	return &State{
		idleArenas:    make(map[string]ArenaServerState),
		runningArenas: make(map[string]ArenaServerState),
		pendingArenas: make(map[string]ArenaServerState),
	}
}
