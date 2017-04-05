package state

import "github.com/netgusto/bytearena/utils"

type ProjectileState struct {
	Position utils.Vector2
	Velocity utils.Vector2
	Ttl      int
	Radius   float64
	From     AgentState
}
