package deathmatch

import "github.com/bytearena/bytearena/common/utils/vector"

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
	MaxSpeed           float64 // max distance covered per turn
	MaxSteeringForce   float64 // max force applied when steering (ie, max magnitude of steering vector)
	MaxAngularVelocity float64
	VisionRadius       float64
	VisionAngle        float64
	DragForce          float64
}

type agentPerceptionVisionItem struct {
	Tag       string
	CloseEdge vector.Vector2
	Center    vector.Vector2
	FarEdge   vector.Vector2
	Velocity  vector.Vector2
}

type agentPerceptionExternal struct {
	Vision []agentPerceptionVisionItem
}

type agentPerceptionInternal struct {
	Energy           float64        // niveau en millièmes; reconstitution automatique ?
	Proprioception   float64        // rayon de la surface occupée par le corps en rayon par rapport au centre géométrique
	Velocity         vector.Vector2 // vecteur de force (direction, magnitude)
	Magnetoreception float64        // azimuth en degrés par rapport au "Nord" de l'arène
}

type agentPerception struct {
	Specs    agentPerceptionSpecs
	External agentPerceptionExternal
	Internal agentPerceptionInternal
}
