package types

import (
	"github.com/bytearena/bytearena/common/utils/vector"
)

type VizMessage struct {
	GameID  string
	Objects []VizMessageObject
}

type VizMessageObject struct {
	Id          string
	Type        string
	Position    vector.Vector2
	Velocity    vector.Vector2
	Radius      float64
	Orientation float64
}
