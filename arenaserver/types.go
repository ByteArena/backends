package arenaserver

type GameStopMessagePayload struct {
	ArenaServerUUID string `json:"arenaserveruuid"`
}

type GameStopMessage struct {
	Payload GameStopMessagePayload `json:"payload"`
}
