package types

import (
	"github.com/bytearena/bytearena/common/utils/vector"
)

type VizMessage struct {
	GameID        string
	Objects       []VizMessageObject
	DebugPoints   [][2]float64
	DebugSegments [][2][2]float64
}

type VizMessageObject struct {
	Id          string
	Type        string
	Position    vector.Vector2
	Velocity    vector.Vector2
	Radius      float64
	Orientation float64
}
