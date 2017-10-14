package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	sourcefilepath := flag.String("in", "", "Input json file; required")
	//pxperunit := flag.Float64("pxperunit", 1.0, "Number of svg px per map unit; default 1.0 (1u = 1px)")
	flag.Parse()

	if *sourcefilepath == "" {
		fmt.Println("--in is required; ex: --in ~/map-playcanvas.json")
		os.Exit(1)
	}

	sourcefile, err := os.Open(*sourcefilepath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer sourcefile.Close()

	source, _ := ioutil.ReadAll(sourcefile)

	pcmodel := loadPlayCanvasModel(source)

	nodesObstacle := make([]*PlayCanvasNode, 0)
	nodesGround := make([]*PlayCanvasNode, 0)
	nodesStart := make([]*PlayCanvasNode, 0)

	for _, node := range pcmodel.nodes {

		if node.MeshInstance == nil { // an empty container, with no mesh attached; skipping
			continue
		}

		nodenames := getNodeNames(node)

		if nodenames.Contains("ba:obstacle") > -1 {
			nodesObstacle = append(nodesObstacle, node)
		}

		if nodenames.Contains("ba:ground") > -1 {
			nodesGround = append(nodesGround, node)
		}

		if nodenames.Contains("ba:start") > -1 {
			nodesStart = append(nodesStart, node)
		}
	}

	//spew.Dump(nodesGround)

	fmt.Println("obstacles", nodesObstacle)
	fmt.Println("grounds", nodesGround)
	fmt.Println("start", nodesStart)

	//grounds := make([]mapcontainer.MapGround, 0)
	for _, groundNode := range nodesGround {
		// Make polygons
		// ground := mapcontainer.MapGround{
		// 	Id:      groundNode.Name,
		// 	Outline: make([]mapcontainer.MapPolygon, 0),
		// }

		vertices := groundNode.MeshInstance.Mesh.GetFlatVertices()
		outlines := outlinesFrom2DMesh(vertices.Position.Points)

		// determining outline

		fmt.Println(outlines)
	}

}

func outlinesFrom2DMesh(points [][3]float64) [][][3]float64 {

	nbtriangles := len(points) / 3
	edges := make([][2][3]float64, 0)

	for trianglenum := 0; trianglenum < nbtriangles; trianglenum++ {
		offset := trianglenum * 3
		edges = append(edges, [2][3]float64{
			points[offset+0],
			points[offset+1],
		})

		edges = append(edges, [2][3]float64{
			points[offset+1],
			points[offset+2],
		})

		edges = append(edges, [2][3]float64{
			points[offset+2],
			points[offset+0],
		})
	}

	//if number.FloatEquals()

	for _, edge := range edges {

	}
}

func getKeyForEdge(edge [2][3]float64) string {

	var min, max = edge[0], edge[1]

	if min[0] > max[0] {
		max, min = min, max
	}

	if min[0] > max[0] {
		return max, min
	}
}

func loadPlayCanvasModel(source []byte) *PlayCanvasModel {
	var pcjson PlayCanvasJSON
	json.Unmarshal(source, &pcjson)

	var pcmodel PlayCanvasModel
	pcmodel.initialize(pcjson)

	return &pcmodel
}

type PlayCanvasModel struct {
	json          PlayCanvasJSON
	nodeTree      *PlayCanvasNode
	nodes         []*PlayCanvasNode
	meshInstances []*PlayCanvasMeshInstance
}

func (model *PlayCanvasModel) initialize(json PlayCanvasJSON) {
	model.json = json

	meshes := make([]*PlayCanvasMesh, len(model.json.Model.Meshes))

	// resolving vertices

	const nbcomponents = 3
	for vertindex, verticesinfo := range model.json.Model.Vertices {
		nbpoints := len(verticesinfo.Normal.Data) / nbcomponents

		normalPoints := make([][nbcomponents]float64, nbpoints)
		positionPoints := make([][nbcomponents]float64, nbpoints)

		for i := 0; i < nbpoints; i++ {
			offset := i * nbcomponents
			normalPoints[i] = [nbcomponents]float64{
				verticesinfo.Normal.Data[offset+0],
				verticesinfo.Normal.Data[offset+1],
				verticesinfo.Normal.Data[offset+2],
			}

			positionPoints[i] = [nbcomponents]float64{
				verticesinfo.Position.Data[offset+0],
				verticesinfo.Position.Data[offset+1],
				verticesinfo.Position.Data[offset+2],
			}
		}

		model.json.Model.Vertices[vertindex].Normal.Points = normalPoints
		model.json.Model.Vertices[vertindex].Position.Points = positionPoints
	}

	// resolving meshes
	for meshindex, meshinfo := range model.json.Model.Meshes {
		mesh := &PlayCanvasMesh{
			AABB:     meshinfo.AABB,
			Vertices: &model.json.Model.Vertices[meshinfo.Vertices],
			Indices:  meshinfo.Indices,
			Count:    meshinfo.Count,
		}

		meshes[meshindex] = mesh
	}

	instances := make([]*PlayCanvasMeshInstance, len(model.json.Model.MeshInstances))

	// resolving Mesh instances
	for instanceindex, instanceinfo := range model.json.Model.MeshInstances {
		instances[instanceindex] = &PlayCanvasMeshInstance{
			Mesh: meshes[instanceinfo.Mesh],
			Node: &model.json.Model.Nodes[instanceinfo.Node],
		}

		// mesh instance associated to node (1 mesh instance => 1 node; 1 node => 0, 1 mesh instance; 1 mesh instance => 1 mesh; 1 mesh => 0, n mesh instances)
		model.json.Model.Nodes[instanceinfo.Node].MeshInstance = instances[instanceindex]
	}

	// resolving nodes
	nodes := make([]*PlayCanvasNode, len(model.json.Model.Nodes))

	for nodeindex, _ := range model.json.Model.Nodes {
		node := &model.json.Model.Nodes[nodeindex]
		node.Children = make([]*PlayCanvasNode, 0)
		nodes[nodeindex] = node
	}

	// building hierarchy

	// on recherche la racine
	var rootNode *PlayCanvasNode
	for nodeIndex, parentIndex := range model.json.Model.Parents {
		if parentIndex == -1 {
			rootNode = &model.json.Model.Nodes[nodeIndex]
			break
		}
	}

	for nodeindex, parentIndex := range model.json.Model.Parents {
		if parentIndex == -1 {
			continue
		}

		var parentNode *PlayCanvasNode

		if parentIndex == 0 {
			parentNode = rootNode
		} else {
			parentNode = nodes[parentIndex]
		}

		childNode := nodes[nodeindex]

		parentNode.Children = append(parentNode.Children, childNode)
		childNode.Parent = parentNode
	}

	model.nodeTree = rootNode
	model.nodes = nodes
}

type PlayCanvasAABB struct {
	Min []float64 `json:"min"`
	Max []float64 `json:"max"`
}

type PlayCanvasEdgeContainer struct {
	Type       string    `json:"type"`
	Components int       `json:"components"`
	Data       []float64 `json:"data"`
	Points     [][3]float64
}

type PlayCanvasVerticeCollection struct {
	Position PlayCanvasEdgeContainer `json:"position"`
	Normal   PlayCanvasEdgeContainer `json:"normal"`
}

type PlayCanvasJSON struct {
	Model struct {
		Version  int                           `json:"version"`
		Nodes    []PlayCanvasNode              `json:"nodes"`
		Parents  []int                         `json:"parents"`
		Vertices []PlayCanvasVerticeCollection `json:"vertices"`
		Meshes   []struct {
			AABB     PlayCanvasAABB `json:"aabb"`
			Vertices int            `json:"vertices"`
			Indices  []int          `json:"indices"`
			Type     string         `json:"type"`
			Base     int            `json:"base"`
			Count    int            `json:"count"`
		}
		MeshInstances []struct {
			Node int `json:"node"`
			Mesh int `json:"mesh"`
		}
	} `json:"model"`
}

type PlayCanvasMesh struct {
	AABB         PlayCanvasAABB
	Vertices     *PlayCanvasVerticeCollection
	Indices      []int
	Count        int
	FlatVertices *PlayCanvasVerticeCollection
}

func (mesh *PlayCanvasMesh) GetFlatVertices() *PlayCanvasVerticeCollection {

	if mesh.FlatVertices != nil {
		return mesh.FlatVertices
	}

	flat := &PlayCanvasVerticeCollection{}
	flat.Position.Points = make([][3]float64, mesh.Count)
	flat.Normal.Points = make([][3]float64, mesh.Count)

	for flatIndex, index := range mesh.Indices {
		flat.Position.Points[flatIndex] = mesh.Vertices.Position.Points[index]
		flat.Normal.Points[flatIndex] = mesh.Vertices.Normal.Points[index]
	}

	return flat
}

type PlayCanvasNode struct {
	Name              string    `json:"name"`
	Position          []float64 `json:"position"`
	Rotation          []float64 `json:"rotation"`
	Scale             []float64 `json:"scale"`
	ScaleCompensation bool      `json:"scaleCompensation"`

	// Node properties not defined as such in PlayCanvas JSON
	Parent       *PlayCanvasNode
	Children     []*PlayCanvasNode
	MeshInstance *PlayCanvasMeshInstance
}

type PlayCanvasMeshInstance struct {
	Mesh *PlayCanvasMesh
	Node *PlayCanvasNode
}

type NodeNameCollection []string

func (c NodeNameCollection) Contains(search string) int {
	for i, group := range c {
		if strings.Contains(group, search) {
			return i
		}
	}

	return -1
}

func getNodeNames(i *PlayCanvasNode) NodeNameCollection {

	var node *PlayCanvasNode = i

	res := make(NodeNameCollection, 0)
	for node != nil {
		if node.Name != "" {
			res = append(res, node.Name)
		}

		node = node.Parent
	}

	return res
}

type NodeNameFunction struct {
	Function string
	Args     json.RawMessage
	Original string
}

func (c NodeNameCollection) GetFunctions() []NodeNameFunction {
	funcs := make([]NodeNameFunction, 0)
	r := regexp.MustCompile("^ba:([a-zA-Z]+)\\((.*?)\\)$")
	for _, group := range c {
		parts := strings.Split(group, "-")
		for _, part := range parts {
			if r.MatchString(part) {
				matches := r.FindStringSubmatch(part)

				funcs = append(funcs, NodeNameFunction{
					Function: matches[1],
					Args:     json.RawMessage("[" + matches[2] + "]"),
					Original: part,
				})
			}
		}
	}

	return funcs
}
