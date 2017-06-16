package types

import (
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type VizMessage struct {
	ArenaId                 string
	Agents                  []VizAgentMessage
	Projectiles             []VizProjectileMessage
	Obstacles               []VizObstacleMessage
	DebugIntersects         []vector.Vector2
	DebugIntersectsRejected []vector.Vector2
	DebugPoints             []vector.Vector2
}

type VizAgentMessage struct {
	Id           uuid.UUID
	Position     vector.Vector2
	Velocity     vector.Vector2
	VisionRadius float64
	VisionAngle  float64
	Radius       float64
	Kind         string
	Orientation  float64
}

type VizProjectileMessage struct {
	Position vector.Vector2
	Radius   float64
	From     VizAgentMessage
	Kind     string
}

type VizObstacleMessage struct {
	Id uuid.UUID
	A  vector.Vector2
	B  vector.Vector2
}
