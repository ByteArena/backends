package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

func main() {

	var fbxdumpCmd string

	switch runtime.GOOS {
	case "darwin":
		{
			fbxdumpCmd = "./bin/fbxdump-macos"
		}
	case "linux":
		{
			fbxdumpCmd = "./bin/fbxdump-linux"
		}
	default:
		{
			fmt.Println("map-builder-fbx may be used only on linux or macos")
			os.Exit(1)
		}
	}

	sourcefilepath := flag.String("in", "", "Input fbx file; required")
	flag.Parse()

	if *sourcefilepath == "" {
		fmt.Println("--in is required; ex: --in ~/map.fbx")
		os.Exit(1)
	}

	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	cmd := exec.Command(
		fbxdumpCmd,
		*sourcefilepath,
	)
	cmd.Env = nil

	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error: error during fbxdump; " + stderr.String())
		os.Exit(1)
	}

	geometries := make(map[int64]*fbxGeometry)
	models := make(map[int64]*fbxModel)

	var f map[string]json.RawMessage
	json.Unmarshal(stdout.Bytes(), &f)

	var topchildren []marshChild
	json.Unmarshal(f["children"], &topchildren)

	for _, topchild := range topchildren {
		if topchild.Name != "Objects" {
			continue
		}
		var children2 []marshChild
		json.Unmarshal(topchild.Children, &children2)
		for _, child2 := range children2 {
			if child2.Name == "Geometry" {

				geometry := fbxGeometry{}
				json.Unmarshal(child2.Properties[0].Value, &geometry.id)
				json.Unmarshal(child2.Properties[1].Value, &geometry.name)

				// cut name up to \null
				geometry.name = strings.Split(geometry.name, "\x00")[0]

				var children3 []marshChild
				json.Unmarshal(child2.Children, &children3)

				for _, child3 := range children3 {
					if child3.Name == "Vertices" {
						json.Unmarshal(child3.Properties[0].Value, &geometry.vertices)
					}

					if child3.Name == "PolygonVertexIndex" {
						json.Unmarshal(child3.Properties[0].Value, &geometry.indices)

						if len(geometry.indices) > 0 {
							poly := make(polygon, 0)
							for _, geometryIndex := range geometry.indices {
								endPoly := false
								if geometryIndex < 0 {
									geometryIndex = geometryIndex*-1 - 1
									endPoly = true
								}

								offset := geometryIndex * 3

								p := point{geometry.vertices[offset+0], geometry.vertices[offset+1], geometry.vertices[offset+2]}
								poly = append(poly, p)

								if endPoly {
									geometry.polygons = append(geometry.polygons, poly)
									poly = make(polygon, 0)
								}
							}

							if len(poly) > 0 {
								geometry.polygons = append(geometry.polygons, poly)
							}
						} else {
							poly := make(polygon, 0)
							for i := 0; i < len(geometry.vertices)/3; i++ {
								offset := i * 3
								p := point{geometry.vertices[offset+0], geometry.vertices[offset+1], geometry.vertices[offset+2]}
								poly = append(poly, p)
							}

							geometry.polygons = append(geometry.polygons, poly)
						}
					}
				}

				geometries[geometry.id] = &geometry

			} else if child2.Name == "Model" {
				model := fbxModel{}
				json.Unmarshal(child2.Properties[0].Value, &model.id)
				json.Unmarshal(child2.Properties[1].Value, &model.name)

				// cut name up to \null
				model.name = strings.Split(model.name, "\x00")[0]

				var children3 []marshChild
				json.Unmarshal(child2.Children, &children3)

				for _, child3 := range children3 {
					if child3.Name == "Properties70" {
						//json.Unmarshal(child3.Properties[0].Value, &geometry.vertices)

						transform := fbxTransform{}
						var children4 []marshChild
						json.Unmarshal(child3.Children, &children4)
						for _, child4 := range children4 {
							if len(child4.Properties) != 7 {
								// always 7 properties on a transform aspect child; example:
								// "properties": [
								// 	{ "type": "S", "value": "Lcl Translation" },
								// 	{ "type": "S", "value": "Lcl Translation" },
								// 	{ "type": "S", "value": "" },
								// 	{ "type": "S", "value": "A" },
								// 	{ "type": "D", "value": 407.526001 },
								// 	{ "type": "D", "value": 578.080200 },
								// 	{ "type": "D", "value": 25.511261 }
								//   ]
								continue
							}

							var kind string
							json.Unmarshal(child4.Properties[0].Value, &kind)

							switch kind {
							case "Lcl Translation":
								{ // position (local translation)
									json.Unmarshal(child4.Properties[4].Value, &transform.translation[0])
									json.Unmarshal(child4.Properties[5].Value, &transform.translation[1])
									json.Unmarshal(child4.Properties[6].Value, &transform.translation[2])
								}
							case "Lcl Rotation":
								{ // position (local rotation)
									json.Unmarshal(child4.Properties[4].Value, &transform.rotation[0])
									json.Unmarshal(child4.Properties[5].Value, &transform.rotation[1])
									json.Unmarshal(child4.Properties[6].Value, &transform.rotation[2])
								}
							case "Lcl Scaling":
								{ // position (local scaling)
									json.Unmarshal(child4.Properties[4].Value, &transform.scaling[0])
									json.Unmarshal(child4.Properties[5].Value, &transform.scaling[1])
									json.Unmarshal(child4.Properties[6].Value, &transform.scaling[2])
								}
							}

						}

						model.transform = transform
					}
				}

				models[model.id] = &model
			}
		}
	}

	for _, topchild := range topchildren {
		if topchild.Name != "Connections" {
			continue
		}

		var children2 []marshChild
		json.Unmarshal(topchild.Children, &children2)
		for _, child2 := range children2 {
			if child2.Name != "C" {
				continue
			}

			var idOne int64
			var idTwo int64

			json.Unmarshal(child2.Properties[1].Value, &idOne)
			json.Unmarshal(child2.Properties[2].Value, &idTwo)

			// Si idOne => model && idTwo empty => model sans parent
			// si idOne => geometry && idTwo model => idTwo.geometry = idOne
			// si idOne => model && idTwo model => idOne.parent = idTwo

			_, idOneIsModel := models[idOne]
			_, idOneIsGeometry := geometries[idOne]
			idTwoIsEmpty := idTwo == 0
			_, idTwoIsModel := models[idTwo]

			if idOneIsModel && idTwoIsEmpty {
				modelOne, _ := models[idOne]
				modelOne.parent = nil
			} else if idOneIsGeometry && idTwoIsModel {
				geometryOne, _ := geometries[idOne]
				modelTwo, _ := models[idTwo]
				modelTwo.geometry = geometryOne
			} else if idOneIsModel && idTwoIsModel {
				modelOne, _ := models[idOne]
				modelTwo, _ := models[idTwo]
				modelOne.parent = modelTwo
				modelTwo.children = append(modelTwo.children, modelOne)
			}
		}
	}

	for _, model := range models {
		spew.Dump(model.name, model.transform, model.geometry)
	}
}

type point [3]float64
type polygon []point

type fbxTransform struct {
	translation [3]float64
	rotation    [3]float64
	scaling     [3]float64
}

type fbxModel struct {
	parent    *fbxModel
	children  []*fbxModel
	id        int64
	name      string
	transform fbxTransform
	geometry  *fbxGeometry
}

type fbxGeometry struct {
	id       int64
	name     string
	vertices []float64
	indices  []int
	polygons []polygon
}

type marshChild struct {
	Name       string          `json:"name"`
	Children   json.RawMessage `json:"children"`
	Properties []marshProperty `json:"properties"`
}

type marshProperty struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}
