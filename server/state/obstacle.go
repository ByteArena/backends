package state

import (
	"github.com/netgusto/bytearena/utils/vector"
)

type Obstacle struct {
	A vector.Vector2
	B vector.Vector2
}

func MakeObstacle(a vector.Vector2, b vector.Vector2) Obstacle {
	return Obstacle{
		A: a,
		B: b,
	}
}
