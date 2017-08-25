package arenaserver

import "github.com/bytearena/bytearena/common/utils/vector"

type GameStopMessagePayload struct {
	ArenaServerUUID string `json:"arenaserveruuid"`
}

type GameStopMessage struct {
	Payload GameStopMessagePayload `json:"payload"`
}

type movingObjectTemporaryState struct {
	Position vector.Vector2
	Velocity vector.Vector2
	Radius   float64
}
