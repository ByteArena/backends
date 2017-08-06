package mapcontainer

import (
	"encoding/json"

	"github.com/bytearena/bytearena/common/utils/number"
)

type MapContainer struct {
	Meta struct {
		Readme         string `json:"readme"`
		Kind           string `json:"kind"`
		Theme          string `json:"theme"`
		MaxContestants int    `json:"maxcontestants"`
		Date           string `json:"date"`
		Repository     string `json:"repository"`
	} `json:"meta"`
	Data struct {
		Grounds   []MapGround   `json:"grounds"`
		Starts    []MapStart    `json:"starts"`
		Obstacles []MapObstacle `json:"obstacles"`
		Objects   []MapObject   `json:"objects"`
	} `json:"data"`
}

type MapPoint struct {
	X float64
	Y float64
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
	Id      string       `json:"id"`
	Outline []MapPolygon `json:"outline"`
	Mesh    Mesh         `json:"mesh"`
}

func MakeMapGround(id string, polygons []MapPolygon) MapGround {
	return MapGround{
		Id:      id,
		Outline: polygons,
		Mesh:    Mesh{},
	}
}

type Mesh struct {
	Vertices []float64 `json:"vertices"`
	Indices  []int     `json:"indices"`
	Uvs      []float64 `json:"uvs"`
}

type MapPolygon struct {
	Points []MapPoint
}

func (p *MapPolygon) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Points)
}

func (a *MapPolygon) UnmarshalJSON(b []byte) error {
	var points []MapPoint
	if err := json.Unmarshal(b, &points); err != nil {
		return err
	}

	a.Points = points

	return nil
}

type MapStart struct {
	Id    string   `json:"id"`
	Point MapPoint `json:"point"`
}

type MapObstacle struct {
	Id      string     `json:"id"`
	Polygon MapPolygon `json:"polygon"`
}

type MapObject struct {
	Id       string   `json:"id"`
	Point    MapPoint `json:"point"`
	Type     string   `json:"type"`
	Diameter float64  `json:"diameter"`
}

/*
{
    "meta": {
        "readme": "Byte Arena Training Map",
        "kind": "deathmatch",
        "theme": "desert",
        "maxcontestants": 2,
        "date": "1234-01-01 00:00:00Z",
        "repository": "http://github.com/bytearena/maps/"
    },
    "data": {
        "grounds": [
            {
                "id": "theground",
                "outline": [
                    [[0, 0], [0, 100], [100, 100], [100, 0], [0, 0]],
                    [[20, 20], [20, 80], [80, 80], [80, 0], [0, 0]]
				],
				"mesh": [
					[[0, 0], [0, 100], [100, 100]],
					[[0, 0], [0, 100], [100, 100]],
					[[0, 0], [0, 100], [100, 100]]
				]
            }
        ],
        "starts": [
            { "id": "one", "point": [[10, 10]] },
            { "id": "two", "point": [[20, 20]] },
            { "id": "three", "point": [[30, 30]] }
        ]
    }
}
*/
