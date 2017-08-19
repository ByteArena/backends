package state

import (
	"github.com/dhconnelly/rtreego"
)

type MapMemoization struct {
	Obstacles      []Obstacle
	RtreeObstacles *rtreego.Rtree
}
