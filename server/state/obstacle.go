package state

import (
	"github.com/netgusto/bytearena/utils"
)

type Obstacle struct {
	A utils.Vector2
	B utils.Vector2
}

func MakeObstacle(a utils.Vector2, b utils.Vector2) Obstacle {
	return Obstacle{
		A: a,
		B: b,
	}
}
