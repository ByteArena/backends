package state

import (
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type Obstacle struct {
	Id     uuid.UUID
	A      vector.Vector2
	B      vector.Vector2
	Normal vector.Vector2
}

func MakeObstacle(a vector.Vector2, b vector.Vector2, normal vector.Vector2) Obstacle {

	relvec := b.Sub(a)
	normalGood := relvec.OrthogonalClockwise().Normalize() // clockwise: because poly is CCW, outwards is on the right of the edge

	return Obstacle{
		Id:     uuid.NewV4(),
		A:      a,
		B:      b,
		Normal: normalGood,
	}
}
