package types

type GameType struct {
	Id          string           `json:"id"`
	Tps         int              `json:"tps"`
	Startedat   string           `json:"startedat"`
	Endedat     string           `json:"endedat"`
	Arena       ArenaType        `json:"arena"`
	Contestants []ContestantType `json:"contestants"`
}
