package protocol

import "github.com/bytearena/bytearena/common/utils/vector"

type _privateAgentPerceptionVisionItemTag string

var AgentPerceptionVisionItemTag = struct {
	Agent      _privateAgentPerceptionVisionItemTag
	Obstacle   _privateAgentPerceptionVisionItemTag
	Projectile _privateAgentPerceptionVisionItemTag
}{
	Agent:      _privateAgentPerceptionVisionItemTag("agent"),
	Obstacle:   _privateAgentPerceptionVisionItemTag("obstacle"),
	Projectile: _privateAgentPerceptionVisionItemTag("projectile"),
}

type AgentPerceptionSpecs struct {
	MaxSpeed           float64 // max distance covered per turn
	MaxSteeringForce   float64 // max force applied when steering (ie, max magnitude of steering vector)
	MaxAngularVelocity float64
	VisionRadius       float64
	VisionAngle        float64
	DragForce          float64
}

type AgentPerceptionVisionItem struct {
	Tag       _privateAgentPerceptionVisionItemTag
	CloseEdge vector.Vector2
	Center    vector.Vector2
	FarEdge   vector.Vector2
	Velocity  vector.Vector2
}

type AgentPerceptionExternal struct {
	Vision []AgentPerceptionVisionItem
}

type AgentPerceptionInternal struct {
	Energy           float64        // niveau en millièmes; reconstitution automatique ?
	Proprioception   float64        // rayon de la surface occupée par le corps en rayon par rapport au centre géométrique
	Velocity         vector.Vector2 // vecteur de force (direction, magnitude)
	Magnetoreception float64        // azimuth en degrés par rapport au "Nord" de l'arène
}

type AgentPerception struct {
	Specs    AgentPerceptionSpecs
	External AgentPerceptionExternal
	Internal AgentPerceptionInternal
}
