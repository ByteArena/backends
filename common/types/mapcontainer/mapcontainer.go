package mapcontainer

import "encoding/json"
import "github.com/bytearena/bytearena/common/utils/number"

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
		Grounds []MapGround `json:"grounds"`
		Starts  []MapStart  `json:"starts"`
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

type MapGround struct {
	Id       string       `json:"id"`
	Polygons []MapPolygon `json:"polygons"`
}

type MapPolygon struct {
	Points []MapPoint
}

func (p *MapPolygon) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Points)
}

type MapStart struct {
	Id    string   `json:"id"`
	Point MapPoint `json:"point"`
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
                "polygons": [
                    [[0, 0], [0, 100], [100, 100], [100, 0], [0, 0]],
                    [[20, 20], [20, 80], [80, 80], [80, 0], [0, 0]]
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
