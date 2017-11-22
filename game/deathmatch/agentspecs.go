package deathmatch

import "github.com/bytearena/bytearena/common/types"

type agentSpecs struct {
	// Movements
	MaxSpeed           float64     `json:"maxspeed"`         // max distance covered per turn
	MaxSteeringForce   float64     `json:"maxsteeringforce"` // max force applied when steering (max length from tip of current velocity vector to tip of next velocity vector)
	MaxAngularVelocity float64     `json:"maxangularvelocity"`
	VisionRadius       float64     `json:"visionradius"`
	VisionAngle        types.Angle `json:"visionangle"`

	// Body
	BodyRadius float64 `json:"bodyradius"`

	// Shoot
	MaxShootEnergy    float64 `json:"maxshootenergy"`
	ShootCost         float64 `json:"shootcost"`
	ShootRecoveryRate float64 `json:"shootrecoveryrate"`
	ShootCooldown     int     `json:"shootcooldown"`
}
