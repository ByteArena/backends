package state

import "github.com/bytearena/bytearena/utils/vector"

type ProjectileState struct {
	Position vector.Vector2
	Velocity vector.Vector2
	Ttl      int
	Radius   float64
	From     AgentState
}
