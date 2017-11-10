package mapcontainer

import (
	"github.com/bytearena/bytearena/common/utils/vector"
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

type MapPoint [2]float64

func MakeMapPointFromVector2(vec vector.Vector2) MapPoint {
	return [2]float64{
		vec.GetX(),
		vec.GetY(),
	}
}

func (m MapPoint) GetX() float64 {
	return m[0]
}

func (m MapPoint) GetY() float64 {
	return m[1]
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
		res = append(res, vector.MakeVector2(point[0], point[1]))
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
