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
	Velocity         vector.Vector2 `json:"velocity"`         // vecteur de force (direction, magnitude)
	Magnetoreception float64        `json:"magnetoreception"` // azimuth en degrés par rapport au "Nord" de l'arène
}

type agentPerception struct {
	External agentPerceptionExternal `json:"external"`
	Internal agentPerceptionInternal `json:"internal"`
}
