package types

import (
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type VizMessage struct {
	GameID          string
	ArenaServerUUID string
	Agents          []VizAgentMessage
	Projectiles     []VizProjectileMessage
	DebugPoints     []vector.Vector2
}

type VizAgentMessage struct {
	Id           uuid.UUID
	Name         string
	Position     vector.Vector2
	Velocity     vector.Vector2
	VisionRadius float64
	VisionAngle  float64
	Radius       float64
	Kind         string
	Orientation  float64
}

type VizProjectileMessage struct {
	Id       uuid.UUID
	Position vector.Vector2
	Velocity vector.Vector2
	Kind     string
}

type VizObstacleMessage struct {
	Id uuid.UUID
	A  vector.Vector2
	B  vector.Vector2
}
