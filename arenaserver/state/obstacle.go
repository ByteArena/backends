package state

import (
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type Obstacle struct {
	Id      uuid.UUID
	segment vector.Segment2
}

func MakeObstacle(s vector.Segment2) Obstacle {
	return Obstacle{
		Id:      uuid.NewV4(),
		segment: s,
	}
}

func (o Obstacle) GetA() vector.Vector2 {
	return o.segment.GetA()
}

func (o Obstacle) GetB() vector.Vector2 {
	return o.segment.GetB()
}
