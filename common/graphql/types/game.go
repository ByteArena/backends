package types

type GameType struct {
	Id            string           `json:"id"`
	Tps           int              `json:"tps"`
	LaunchedAt    string           `json:"launchedAt"`
	EndedAt       string           `json:"endedAt"`
	Arena         *ArenaType       `json:"arena"`
	Contestants   []ContestantType `json:"contestants"`
	ArenaServerId string           `json:"arenaServerId"`
	RunStatus     int              `json:"runStatus"`
	RunError      string           `json:"runError"`
}
