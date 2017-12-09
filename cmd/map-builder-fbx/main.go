package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-gl/mathgl/mgl64"

	"github.com/bytearena/core/common/types/mapcontainer"
	"github.com/bytearena/core/common/utils/number"
	polygonutils "github.com/bytearena/core/common/utils/polygon"
	"github.com/bytearena/core/common/utils/vector"
)

const AxisX = 0
const AxisY = 1
const AxisZ = 2

// 					Blender		Sign			FBX				Sign

// CoordAxis		AxisX = 0	1				AxisX = 0		1
// UpAxis			AxisZ = 2	1				AxisY = 1		1
// FrontAxis		AxisY = 1	-1				AxisZ = 2		1

// => For blender, FrontAxis is 1 when pointing away from the camera, and -1 when pointing towards the camera; it's the opposite for FBX
// => yet blenders does not set the sign to -1 on FrontAxis when exporting; see https://developer.blender.org/T43935 ?

// 					(1,0,0)			(1,0,0)
// 					(0,1,0)			(0,0,-1)
//					(0,0,1)			(0,1,0)

// => -z (fbx) becomes y (corrected vertice)
// => y (fbx) becomes z

func fixCoordSystem(p vertexType) vertexType {
	return vertexType{
		p[0],
		-1.0 * p[2],
		p[1],
	}
}

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

	// fmt.Println(stdout)
	// os.Exit(0)

	geometries := make(map[int64]*fbxGeometry)
	models := make(map[int64]*fbxModel)

	var f map[string]json.RawMessage
	json.Unmarshal(stdout.Bytes(), &f)

	var topchildren []marshChild
	json.Unmarshal(f["children"], &topchildren)

	scene := fbxScene{}

	for _, topchild := range topchildren {
		if topchild.Name == "GlobalSettings" {
			var children2 []marshChild
			json.Unmarshal(topchild.Children, &children2)
			for _, child2 := range children2 {
				if child2.Name == "Properties70" {
					var children3 []marshChild
					json.Unmarshal(child2.Children, &children3)

					for _, child3 := range children3 {
						if child3.Name == "P" {
							var propname string
							json.Unmarshal(child3.Properties[0].Value, &propname)

							var valuePointer *int

							switch propname {
							case "UpAxis":
								valuePointer = &scene.upAxis
							case "UpAxisSign":
								valuePointer = &scene.upAxisSign
							case "FrontAxis":
								valuePointer = &scene.frontAxis
							case "FrontAxisSign":
								valuePointer = &scene.frontAxisSign
							case "CoordAxis":
								valuePointer = &scene.coordsAxis
							case "CoordAxisSign":
								valuePointer = &scene.coordsAxisSign
							default:
								continue
							}

							var value int
							json.Unmarshal(child3.Properties[4].Value, &value)
							*valuePointer = value
						}
					}
				}
			}
		}

		if topchild.Name == "Objects" {
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

								poly := make(faceType, 0)

								for _, geometryIndex := range geometry.indices {
									endPoly := false
									if geometryIndex < 0 {
										// https://www.scratchapixel.com/lessons/3d-basic-rendering/introduction-polygon-mesh/polygon-mesh-file-formats
										geometryIndex = geometryIndex*-1 - 1
										endPoly = true
									}

									offset := geometryIndex * 3

									p := vertexType{geometry.vertices[offset+0], geometry.vertices[offset+1], geometry.vertices[offset+2]}
									poly = append(poly, fixCoordSystem(p))

									if endPoly {
										geometry.faces = append(geometry.faces, poly)
										poly = make(faceType, 0)
									}
								}

								if len(poly) > 0 {
									geometry.faces = append(geometry.faces, poly)
								}
							} else {
								poly := make(faceType, 0)
								for i := 0; i < len(geometry.vertices)/3; i++ {
									offset := i * 3
									p := vertexType{geometry.vertices[offset+0], geometry.vertices[offset+1], geometry.vertices[offset+2]}
									poly = append(poly, fixCoordSystem(p))
								}

								geometry.faces = append(geometry.faces, poly)
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

										transform.translation = fixCoordSystem(transform.translation)
									}
								case "Lcl Rotation":
									{ // position (local rotation)
										json.Unmarshal(child4.Properties[4].Value, &transform.rotation[0])
										json.Unmarshal(child4.Properties[5].Value, &transform.rotation[1])
										json.Unmarshal(child4.Properties[6].Value, &transform.rotation[2])

										transform.rotation = fixCoordSystem(transform.rotation)
									}
								case "Lcl Scaling":
									{ // position (local scaling)
										json.Unmarshal(child4.Properties[4].Value, &transform.scaling[0])
										json.Unmarshal(child4.Properties[5].Value, &transform.scaling[1])
										json.Unmarshal(child4.Properties[6].Value, &transform.scaling[2])

										transform.scaling = fixCoordSystem(transform.scaling)
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

	modelsObstacle := make([]*fbxModel, 0)
	modelsGround := make([]*fbxModel, 0)
	modelsStart := make([]*fbxModel, 0)

	for _, model := range models {
		if model.geometry == nil {
			continue
		}

		modelnames := getNodeNames(model)

		if modelnames.Contains("ba:obstacle") > -1 {
			modelsObstacle = append(modelsObstacle, model)
		}

		if modelnames.Contains("ba:ground") > -1 {
			modelsGround = append(modelsGround, model)
		}

		if modelnames.Contains("ba:start") > -1 {
			modelsStart = append(modelsStart, model)
		}
	}

	grounds := make([]mapcontainer.MapGround, 0)
	obstacles := make([]mapcontainer.MapObstacleObject, 0)
	starts := make([]mapcontainer.MapStart, 0)

	for _, model := range modelsObstacle {
		//fmt.Println("# " + model.name)
		obstacles = append(obstacles, mapcontainer.MapObstacleObject{
			Id:   strconv.Itoa(int(model.id)),
			Name: model.name,
			Polygon: polygonFrom2DMesh(
				model.geometry.getTransformedFaces(model.getFullTransform()),
			),
		})
	}

	for _, model := range modelsGround {
		//fmt.Println("# " + model.name)
		grounds = append(grounds, mapcontainer.MapGround{
			Id:   strconv.Itoa(int(model.id)),
			Name: model.name,
			Polygon: polygonFrom2DMesh(
				model.geometry.getTransformedFaces(model.getFullTransform()),
			),
		})
	}

	for _, start := range modelsStart {
		origin := vertexType{0, 0, 0}.applyTransform(start.getFullTransform())
		starts = append(starts, mapcontainer.MapStart{
			Id:   strconv.Itoa(int(start.id)),
			Name: start.name,
			Point: mapcontainer.MapPoint{
				origin[0],
				origin[1],
			},
		})
	}

	builtmap := mapcontainer.MapContainer{}

	builtmap.Meta.Readme = "Byte Arena Map"
	builtmap.Meta.Kind = "deathmatch"
	builtmap.Meta.MaxContestants = len(starts)
	builtmap.Meta.Date = time.Now().Format(time.RFC3339)

	builtmap.Data.Grounds = grounds
	builtmap.Data.Starts = starts
	builtmap.Data.Obstacles = obstacles

	bjsonmap, _ := json.MarshalIndent(builtmap, "", "    ")
	fmt.Println(string(bjsonmap))
}

func debugPolygonSVG(polygon []vector.Vector2) {
	fmt.Println("<svg height='700' width='500'><g transform='translate(700,350) scale(4)'>")
	for i := 0; i < len(polygon); i++ {
		nextIndex := 0
		if i == len(polygon)-1 {
			nextIndex = 0
		} else {
			nextIndex = i + 1
		}
		a := polygon[i]
		b := polygon[nextIndex]
		fmt.Println(fmt.Sprintf("<line title='Edge #%d' x1=\"%f\" y1=\"%f\" x2=\"%f\" y2=\"%f\" style=\"stroke: black; stroke-width: 0.1;\" />", i+1, a.GetX()*10, a.GetY()*10, b.GetX()*10, b.GetY()*10))
	}

	fmt.Println("</g></svg>")
}

type vertexType [3]float64
type edgeType [2]vertexType
type faceType []vertexType

func (a vertexType) Equals(b vertexType) bool {
	return number.FloatEquals(a[0], b[0]) && number.FloatEquals(a[1], b[1]) && number.FloatEquals(a[2], b[2])
}

func (p vertexType) String() string {
	return fmt.Sprintf("<vertex(%f, %f, %f)>", p[0], p[1], p[2])
}

func (p vertexType) applyTransform(transform mgl64.Mat4) vertexType {
	res := mgl64.TransformCoordinate(
		mgl64.Vec3{
			p[0],
			p[1],
			p[2],
		},
		transform,
	)

	return vertexType{
		res.X(),
		res.Y(),
		res.Z(),
	}
}

func (a edgeType) Equals(b edgeType) bool {
	return a[0].Equals(b[0]) && a[1].Equals(b[1]) || a[1].Equals(b[0]) && a[0].Equals(b[1])
}

func (face faceType) GetEdges() []edgeType {
	edges := make([]edgeType, 0)

	for i := 0; i < len(face); i++ {

		nextIndex := i + 1
		if i == len(face)-1 {
			nextIndex = 0
		}

		edges = append(edges, edgeType{face[i], face[nextIndex]})
	}

	return edges
}

func (poly faceType) applyTransform(transform mgl64.Mat4) faceType {
	res := make(faceType, len(poly))

	for i, p := range poly {
		res[i] = p.applyTransform(transform)
	}

	return res
}

type fbxScene struct {
	upAxis         int
	upAxisSign     int
	frontAxis      int
	frontAxisSign  int
	coordsAxis     int
	coordsAxisSign int
}

type fbxTransform struct {
	translation vertexType
	rotation    vertexType
	scaling     vertexType
}

type fbxModel struct {
	parent    *fbxModel
	children  []*fbxModel
	id        int64
	name      string
	transform fbxTransform
	geometry  *fbxGeometry
}

func (model *fbxModel) getFullTransform() mgl64.Mat4 {

	// ordre : local -> global

	mats := make([]mgl64.Mat4, 0)

	mats = append(mats, mgl64.Scale3D(0.01, 0.01, 0.01))

	current := model
	for current != nil {

		var scale mgl64.Mat4

		if current.transform.scaling[0] == 0.0 && current.transform.scaling[1] == 0.0 && current.transform.scaling[2] == 0.0 {
			scale = mgl64.Scale3D(1, 1, 1)
		} else {
			scale = mgl64.Scale3D(current.transform.scaling[0], current.transform.scaling[1], current.transform.scaling[2])
		}

		rotx := mgl64.HomogRotate3DX(mgl64.DegToRad(current.transform.rotation[0]))
		roty := mgl64.HomogRotate3DY(mgl64.DegToRad(current.transform.rotation[1]))
		rotz := mgl64.HomogRotate3DZ(mgl64.DegToRad(current.transform.rotation[2]))

		trans := mgl64.Translate3D(current.transform.translation[0]/100.0, current.transform.translation[1]/100.0, current.transform.translation[2]/100.0)

		mat := mgl64.Ident4().
			Mul4(trans).
			Mul4(rotz).
			Mul4(roty).
			Mul4(rotx).
			Mul4(scale)

		mats = append(mats, mat)

		current = current.parent
	}

	mat := mgl64.Ident4()

	for i := len(mats) - 1; i >= 0; i-- {
		mat = mat.Mul4(mats[i])
	}

	return mat
}

type fbxGeometry struct {
	id       int64
	name     string
	vertices []float64
	indices  []int
	faces    []faceType
}

func (g *fbxGeometry) getTransformedFaces(transform mgl64.Mat4) []faceType {
	res := make([]faceType, len(g.faces))
	for i, face := range g.faces {
		res[i] = face.applyTransform(transform)
	}

	return res
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

///////////////////////////////////////////////////////////////////////////////

func makeSortedEdge(a, b vertexType) edgeType {

	var min, max = a, b

	if min[0] > max[0] {
		max, min = min, max
	} else if min[0] == max[0] && min[1] > max[1] {
		max, min = min, max
	} else if min[0] == max[0] && min[1] == max[1] && min[2] > max[2] {
		max, min = min, max
	}

	return edgeType{
		min,
		max,
	}
}

func polygonFrom2DMesh(faces []faceType) mapcontainer.MapPolygon {

	edges := make([]edgeType, 0)
	polygon := make([]vector.Vector2, 0)

	for _, face := range faces {
		edges = append(edges, face.GetEdges()...)
	}

	// sort edges
	sortedEdges := make([]edgeType, len(edges))
	for i, edge := range edges {
		sortedEdges[i] = makeSortedEdge(edge[0], edge[1])
	}

	type edgecount struct {
		count int
		edge  edgeType
	}

	countedEdges := make(map[string]*edgecount)

	for _, edge := range sortedEdges {
		hash := getKeyForEdge(edge)
		//fmt.Println("HASH", hash)

		_, ok := countedEdges[hash]
		if !ok {
			countedEdges[hash] = &edgecount{
				count: 1,
				edge:  edge,
			}
		} else {
			countedEdges[hash].count++
		}
	}

	//spew.Dump(points)
	//spew.Dump(countedEdges)

	outlineEdges := make([]edgeType, 0)

	// stabilizing map iteration (easier for debug)
	var countedEdgesKeys []string
	for k := range countedEdges {
		countedEdgesKeys = append(countedEdgesKeys, k)
	}

	sort.Strings(countedEdgesKeys)

	for _, countedEdgeKey := range countedEdgesKeys {
		countedEdge := countedEdges[countedEdgeKey]
		if countedEdge.count == 1 {
			outlineEdges = append(outlineEdges, countedEdge.edge)
		}
	}

	if len(outlineEdges) == 0 {
		return mapcontainer.MapPolygon{}
	}

	/////////////////////////////////////////////////////////////
	// putting edges in the right order for the polygon
	/////////////////////////////////////////////////////////////

	outline := make([]edgeType, 0)

	// taking the leftmost point as a starting point
	var leftMostEdge *edgeType
	for _, edge := range outlineEdges {
		if leftMostEdge == nil || leftMostEdge[0][0] > edge[0][0] {
			leftMostEdge = &edge
		}
	}

	outline = append(outline, *leftMostEdge)
	done := false
	for i := 1; i < len(outlineEdges); i++ {
		head := outline[i-1]
		found := false
		for _, edge := range outlineEdges {
			if head.Equals(edge) {
				continue
			}

			if head[1].Equals(edge[0]) {
				outline = append(outline, edge)
				found = true
				done = edge.Equals(outline[0])
				break
			} else if head[1].Equals(edge[1]) {
				// swap edge points
				edge[0], edge[1] = edge[1], edge[0]
				outline = append(outline, edge)
				found = true
				done = edge.Equals(outline[0])
				break
			}
		}

		if !found {
			fmt.Println("Next edge not found in outlinesFrom2DMesh for", head, outlineEdges)
			//os.Exit(1)
			break
		}

		if done {
			break
		}
	}

	// /////////////////////////////////////////////////////////////
	// /////////////////////////////////////////////////////////////

	// // convert edges to points (vector2)
	for _, edge := range outline {
		polygon = append(polygon, vector.MakeVector2(edge[0][0], edge[0][1]))
	}

	// // ensure winding
	polygon, err := polygonutils.EnsureWinding(polygonutils.CartesianSystemWinding.CCW, polygon)
	if err != nil {
		fmt.Println(err)
		return mapcontainer.MapPolygon{}
	}

	points := make([]mapcontainer.MapPoint, 0)
	for _, vec2 := range polygon {
		points = append(points, mapcontainer.MapPoint{vec2.GetX(), vec2.GetY()})
	}

	return mapcontainer.MapPolygon{Points: points}
}

func getKeyForEdge(edge edgeType) string {
	return fmt.Sprintf("%.5f_%.5f_%.5f_%.5f", edge[0][0], edge[0][1], edge[1][0], edge[1][1])
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type ModelNameCollection []string

func (c ModelNameCollection) Contains(search string) int {

	for i, group := range c {
		if strings.Contains(group, search) {
			return i
		}
	}

	return -1
}

func getNodeNames(i *fbxModel) ModelNameCollection {

	var model *fbxModel = i

	res := make(ModelNameCollection, 0)
	for model != nil {
		if model.name != "" {
			res = append(res, model.name)
		}

		model = model.parent
	}

	return res
}

type ModelNameFunction struct {
	Function string
	Args     json.RawMessage
	Original string
}

func (c ModelNameCollection) GetFunctions() []ModelNameFunction {
	funcs := make([]ModelNameFunction, 0)
	r := regexp.MustCompile("^ba:([a-zA-Z]+)\\((.*?)\\)$")
	for _, group := range c {
		parts := strings.Split(group, "-")
		for _, part := range parts {
			if r.MatchString(part) {
				matches := r.FindStringSubmatch(part)

				funcs = append(funcs, ModelNameFunction{
					Function: matches[1],
					Args:     json.RawMessage("[" + matches[2] + "]"),
					Original: part,
				})
			}
		}
	}

	return funcs
}
