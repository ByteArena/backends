package deathmatch

import "github.com/bytearena/bytearena/common/utils/vector"

type Angle float64

var agentPerceptionVisionItemTag = struct {
	Agent      string
	Obstacle   string
	Projectile string
}{
	Agent:      "agent",
	Obstacle:   "obstacle",
	Projectile: "projectile",
}

type agentPerceptionSpecs struct {
	MaxSpeed           float64 `json:"maxspeed"`         // max distance covered per turn
	MaxSteeringForce   float64 `json:"maxsteeringforce"` // max force applied when steering (ie, max magnitude of steering vector)
	MaxAngularVelocity float64 `json:"maxangularvelocity"`
	VisionRadius       float64 `json:"visionradius"`
	VisionAngle        Angle   `json:"visionangle"`
	DragForce          float64 `json:"dragforce"`
}

type agentPerceptionVisionItem struct {
	Tag      string         `json:"tag"`
	NearEdge vector.Vector2 `json:"nearedge"`
	Center   vector.Vector2 `json:"center"`
	FarEdge  vector.Vector2 `json:"faredge"`
	Velocity vector.Vector2 `json:"velocity"`
}

type agentPerceptionExternal struct {
	Vision []agentPerceptionVisionItem `json:"vision"`
}

type agentPerceptionInternal struct {
	Energy           float64        `json:"energy"`           // niveau en millièmes; reconstitution automatique ?
	Proprioception   float64        `json:"proprioception"`   // rayon de la surface occupée par le corps en rayon par rapport au centre géométrique
	Velocity         vector.Vector2 `json:"velocity"`         // vecteur de force (direction, magnitude)
	Magnetoreception float64        `json:"magnetoreception"` // azimuth en degrés par rapport au "Nord" de l'arène
}

type agentPerception struct {
	Specs    agentPerceptionSpecs    `json:"specs"`
	External agentPerceptionExternal `json:"external"`
	Internal agentPerceptionInternal `json:"internal"`
}
