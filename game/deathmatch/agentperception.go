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
	MaxSpeed           float64 `json:"maxSpeed"`         // max distance covered per turn
	MaxSteeringForce   float64 `json:"maxSteeringForce"` // max force applied when steering (ie, max magnitude of steering vector)
	MaxAngularVelocity float64 `json:"maxAngularVelocity"`
	VisionRadius       float64 `json:"visionRadius"`
	VisionAngle        float64 `json:"visionangle"`
	DragForce          float64 `json:"dragForce"`
}

type agentPerceptionVisionItem struct {
	Tag       _privateAgentPerceptionVisionItemTag `json:"tag"`
	CloseEdge vector.Vector2                       `json:"closeEdge"`
	Center    vector.Vector2                       `json:"center"`
	FarEdge   vector.Vector2                       `json:"farEdge"`
	Velocity  vector.Vector2                       `json:"velocity"`
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
