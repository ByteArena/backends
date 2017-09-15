package state

import (
	"github.com/bytearena/bytearena/common/utils/vector"
)

var ObstacleType = struct {
	Ground int
	Object int
}{
	Ground: 1,
	Object: 2,
}

type Obstacle struct {
	Id     string
	Type   int
	A      vector.Vector2
	B      vector.Vector2
	Normal vector.Vector2
}

func MakeObstacle(id string, obstacletype int, a vector.Vector2, b vector.Vector2, normal vector.Vector2) Obstacle {

	relvec := b.Sub(a)
	normalGood := relvec.OrthogonalClockwise().Normalize() // clockwise: because poly is CCW, outwards is on the right of the edge

	return Obstacle{
		Id:     id,
		Type:   obstacletype,
		A:      a,
		B:      b,
		Normal: normalGood,
	}
}
