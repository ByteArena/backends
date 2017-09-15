package state

import (
	"github.com/dhconnelly/rtreego"
)

type MapMemoization struct {
	Obstacles       []Obstacle
	RtreeObstacles  *rtreego.Rtree
	RtreeSurface    *rtreego.Rtree
	RtreeCollisions *rtreego.Rtree
	RtreeMoving     *rtreego.Rtree
}
