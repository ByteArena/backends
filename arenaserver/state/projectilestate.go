package state

import "github.com/bytearena/bytearena/common/utils/vector"

import "github.com/bytearena/bytearena/arenaserver/state/protocol"

type ProjectileState struct {
	Position vector.Vector2
	Velocity vector.Vector2
	Ttl      int
	Radius   float64
	From     protocol.AgentStateInterface
}
