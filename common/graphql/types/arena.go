package types

type ArenaType struct {
	Id             int            `json:"id"`
	Name           string         `json:"name"`
	Kind           string         `json:"kind"`
	Maxcontestants int            `json:"maxcontestants"`
	Surface        SurfaceType    `json:"surface"`
	Obstacles      []ObstacleType `json:"obstacles"`
}
