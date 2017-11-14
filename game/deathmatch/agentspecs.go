package deathmatch

import "github.com/bytearena/bytearena/common/types"

type agentSpecs struct {
	MaxSpeed           float64     `json:"maxspeed"`         // max distance covered per turn
	MaxSteeringForce   float64     `json:"maxsteeringforce"` // max force applied when steering (ie, max magnitude of steering vector)
	MaxAngularVelocity float64     `json:"maxangularvelocity"`
	VisionRadius       float64     `json:"visionradius"`
	VisionAngle        types.Angle `json:"visionangle"`
	BodyRadius         float64     `json:"bodyradius"`
}
