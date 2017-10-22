package mapcontainer

import (
	"encoding/json"

	"github.com/bytearena/bytearena/common/utils/vector"

	"github.com/bytearena/bytearena/common/utils/number"
)

type MapContainer struct {
	Meta struct {
		Readme         string `json:"readme"`
		Kind           string `json:"kind"`
		MaxContestants int    `json:"maxcontestants"`
		Date           string `json:"date"`
		Repository     string `json:"repository"`
	} `json:"meta"`
	Data struct {
		Grounds         []MapGround         `json:"grounds"`
		Starts          []MapStart          `json:"starts"`
		Obstacles       []MapObstacleObject `json:"obstacles"`
		CollisionMeshes []CollisionMesh     `json:"collisionmeshes"`
	} `json:"data"`
}

type MapPoint struct {
	X float64
	Y float64
}

func MakeMapPointFromVector2(vec vector.Vector2) MapPoint {
	return MapPoint{
		X: vec.GetX(),
		Y: vec.GetY(),
	}
}

func (p *MapPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{
		number.ToFixed(p.X, 5),
		number.ToFixed(p.Y, 5),
	})
}

func (a *MapPoint) UnmarshalJSON(b []byte) error {
	var floats []float64
	if err := json.Unmarshal(b, &floats); err != nil {
		return err
	}

	a.X = floats[0]
	a.Y = floats[1]

	return nil
}

type MapGround struct {
	Id      string     `json:"id"`
	Name    string     `json:"name"`
	Polygon MapPolygon `json:"polygon"`
	Mesh    Mesh       `json:"mesh"`
}

type Mesh struct {
	Vertices []float64 `json:"vertices"`
	Indices  []int     `json:"indices"`
	Uvs      []float64 `json:"uvs"`
}

type CollisionMesh struct {
	Id       string    `json:"id"`
	Vertices []float64 `json:"vertices"`
}

type MapPolygon struct {
	Points []MapPoint `json:"points"`
}

func (a *MapPolygon) ToVector2Array() []vector.Vector2 {
	res := make([]vector.Vector2, 0)
	for _, point := range a.Points {
		res = append(res, vector.MakeVector2(point.X, point.Y))
	}

	return res
}

type MapStart struct {
	Id    string   `json:"id"`
	Name  string   `json:"name"`
	Point MapPoint `json:"point"`
}

type MapObstacleObject struct {
	Id      string     `json:"id"`
	Name    string     `json:"name"`
	Polygon MapPolygon `json:"polygon"`
}
