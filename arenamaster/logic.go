package arenamaster

import (
	"strconv"
)

func getMasterStatus(state *State) string {
	return "(" + strconv.Itoa(len(state.idleArenas)) + " arena(s) idle, " + strconv.Itoa(len(state.runningArenas)) + " arena(s) running, " + strconv.Itoa(len(state.pendingArenas)) + " arena(s) pending)"
}
