package state

import (
	"github.com/bytearena/bytearena/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type Obstacle struct {
	Id uuid.UUID
	A  vector.Vector2
	B  vector.Vector2
}

func MakeObstacle(a vector.Vector2, b vector.Vector2) Obstacle {
	return Obstacle{
		Id: uuid.NewV4(),
		A:  a,
		B:  b,
	}
}
