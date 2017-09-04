package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/number"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/vector"
	poly2tri "github.com/netgusto/poly2tri-go"

	polygonutils "github.com/bytearena/bytearena/cmd/map-builder/polygon"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	source := flag.String("in", "", "Input svg file; required")
	pxperunit := flag.Float64("pxperunit", 1.0, "Number of svg px per map unit; default 1.0 (1u = 1px)")
	flag.Parse()

	if *source == "" {
		fmt.Println("--in is required; ex: --in ~/map.svg")
		os.Exit(1)
	}

	svgsource, err := os.Open(*source)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer svgsource.Close()

	b, _ := ioutil.ReadAll(svgsource)

	svg := ParseSVG(b)

	//fmt.Println(SVGDebug(svg, 0))

	builtmap := buildMap(svg, *pxperunit)
	bjsonmap, _ := json.MarshalIndent(builtmap, "", "")

	fmt.Println(strings.Replace(string(bjsonmap), "\n", "", -1))
}

func buildMap(svg SVGNodeInterface, pxperunit float64) mapcontainer.MapContainer {
	// Path + Fill=groundcolor: ground
	// Circle/Ellipse + Fill=red: starting spot
	// Circle/Ellipse + Fill=obstaclecolor: normalized obstacle
	// Polygon + Fill=obstaclecolor: custom obstacle

	worldTransform := vector.MakeMatrix2()
	if pxperunit != 1.0 {
		worldTransform = worldTransform.Scale(1/pxperunit, 1/pxperunit)
	}

	// Convert coordinate system of SVG into the one of BabylonJS
	/*

		SVG:

		o--------- x
		|
		|
		|
		y


		BabylonJS:

		y
		|   z
		|  /
		| /
		o---------x

		As a consequence:

		BabylonJS(x) <= SVG(x)
		BabylonJS(z) <= -1 * SVG(y)
		BabylonJS(y) <= 0 (we're handling 2D polygon, so this dimension is not specified)

	*/

	worldTransform = worldTransform.Scale(1, -1)

	svggrounds := make([]SVGNodeInterface, 0)
	svgstarts := make([]SVGNodeInterface, 0)
	svgobjects := make([]SVGNodeInterface, 0)
	svgobstacles := make([]SVGNodeInterface, 0)

	SVGVisit(svg, func(node SVGNodeInterface) {

		groups := GetSVGIDs(node)

		switch /*typednode :=*/ node.(type) {
		case *SVGPath:
			{
				if groups.Contains("ba:ground") > -1 {
					svggrounds = append(svggrounds, node)
				}
			}
		case *SVGCircle:
			{
				if groups.Contains("ba:start") > -1 {
					svgstarts = append(svgstarts, node)
				} else {
					if groups.Contains("ba:obstacle") > -1 {
						svgobstacles = append(svgobstacles, node)
					}

					if groups.Contains("ba:object") > -1 {
						svgobjects = append(svgobjects, node)
					}
				}
			}
		case *SVGEllipse:
			{
				if groups.Contains("ba:start") > -1 {
					svgstarts = append(svgstarts, node)
				} else {
					if groups.Contains("ba:obstacle") > -1 {
						svgobstacles = append(svgobstacles, node)
					}

					if groups.Contains("ba:object") > -1 {
						svgobjects = append(svgobjects, node)
					}
				}
			}
		case *SVGPolygon:
			{
				if groups.Contains("ba:ground") > -1 {
					svggrounds = append(svggrounds, node)
				}

				if groups.Contains("ba:obstacle") > -1 {
					svgobstacles = append(svgobstacles, node)
				}
			}
		}
	})

	/************************************/
	/* Processing grounds */
	/************************************/

	grounds := processGrounds(worldTransform, svggrounds)

	/************************************/
	/* Processing STARTS */
	/************************************/

	starts := processStarts(worldTransform, svgstarts)

	/************************************/
	/* Processing OBJECTS */
	/************************************/

	objects, _ := processObjects(worldTransform, svgobjects)

	/************************************/
	/* Processing OBSTACLES */
	/************************************/
	// use processObjects to get a pre-processed collection of objects, that we will convert to obstacles
	obstaclesPrefabs, obstaclesPolygons := processObjects(worldTransform, svgobstacles)
	obstacles := make([]mapcontainer.MapObstacleObject, 0)

	// Obstacle prefabs

	for _, obstacleObject := range obstaclesPrefabs {

		switch obstacleObject.Type {
		case "rocks01", "rocks02", "rocksRand", "alienBones", "satellite01", "satellite02", "crater01", "crater02", "craterRand", "station01", "station02", "stationRand":
			{
				// temporary: draw a square around the center of the obstacle
				//
				// A---B
				// | o |	// o: center
				// D---C

				center := obstacleObject.Point
				diameter := obstacleObject.Diameter
				halfDiameter := diameter / 2.0

				polyPoints, _ := polygonutils.EnsureWinding(polygonutils.CartesianSystemWinding.CCW, []vector.Vector2{
					vector.MakeVector2(
						center.X-halfDiameter,
						center.Y-halfDiameter,
					),
					vector.MakeVector2(
						center.X+halfDiameter,
						center.Y-halfDiameter,
					),
					vector.MakeVector2(
						center.X+halfDiameter,
						center.Y+halfDiameter,
					),
					vector.MakeVector2(
						center.X-halfDiameter,
						center.Y+halfDiameter,
					),
				})

				normals := make([]mapcontainer.MapPoint, 0)
				polylen := len(polyPoints)
				for i := 0; i < polylen; i++ {
					p1 := polyPoints[i]
					p2 := polyPoints[(i+1)%polylen]
					vec := vector.MakeVector2(p2.GetX(), p2.GetY()).Sub(p1)
					normal := vec.OrthogonalClockwise().Normalize() // clockwise: because poly is CCW, outwards is on the right of the edge
					normals = append(normals, mapcontainer.MapPoint{
						X: normal.GetX(),
						Y: normal.GetY(),
					})
				}

				polygon := mapcontainer.MapPolygon{
					Points: []mapcontainer.MapPoint{
						mapcontainer.MakeMapPointFromVector2(polyPoints[0]),
						mapcontainer.MakeMapPointFromVector2(polyPoints[1]),
						mapcontainer.MakeMapPointFromVector2(polyPoints[2]),
						mapcontainer.MakeMapPointFromVector2(polyPoints[3]),
						mapcontainer.MakeMapPointFromVector2(polyPoints[0]),
					},
					Normals: normals,
				}

				obstacles = append(obstacles, mapcontainer.MapObstacleObject{
					Id:      obstacleObject.Id,
					Polygon: polygon,
				})
				break
			}
		default:
			{
				utils.Debug("map-builder", "Unsuported obstacle type "+obstacleObject.Type)
				os.Exit(1)
			}
		}
	}

	/************************************/
	/* Building collision meshes
	/************************************/

	obstacles = append(obstacles, obstaclesPolygons...)

	collisionMeshes := make([]mapcontainer.CollisionMesh, 0)

	for _, obstacle := range obstacles {
		contour := make([]*poly2tri.Point, 0)

		firstPoint := obstacle.Polygon.Points[0]
		lastPoint := obstacle.Polygon.Points[len(obstacle.Polygon.Points)-1]

		sliceLength := len(obstacle.Polygon.Points)

		if firstPoint.X == lastPoint.X && firstPoint.Y == lastPoint.Y {
			sliceLength -= 1
		}

		for _, point := range obstacle.Polygon.Points[:sliceLength] {
			contour = append(contour, poly2tri.NewPoint(point.X, point.Y))
		}

		swctx := poly2tri.NewSweepContext(contour, false)
		swctx.Triangulate()
		triangles := swctx.GetTriangles()

		vertices := make([]float64, 0)
		for _, triangle := range triangles {
			vertices = append(vertices, []float64{
				triangle.Points[0].GetX(), 0, triangle.Points[0].GetY(),
				triangle.Points[1].GetX(), 0, triangle.Points[1].GetY(),
				triangle.Points[2].GetX(), 0, triangle.Points[2].GetY(),
			}...)
		}

		collisionMeshes = append(collisionMeshes, mapcontainer.CollisionMesh{
			Id:       obstacle.Id,
			Vertices: vertices,
		})
	}

	/************************************/
	/************************************/

	builtmap := mapcontainer.MapContainer{}

	builtmap.Meta.Readme = "Byte Arena Training Map"
	builtmap.Meta.Kind = "deathmatch"
	builtmap.Meta.Theme = "desert"
	builtmap.Meta.MaxContestants = 2
	builtmap.Meta.Date = time.Now().Format(time.RFC3339)
	builtmap.Meta.Repository = "https://github.com/bytearena/maps"

	builtmap.Data.Grounds = grounds
	builtmap.Data.Starts = starts
	builtmap.Data.Objects = objects
	builtmap.Data.Obstacles = obstacles
	builtmap.Data.CollisionMeshes = collisionMeshes

	return builtmap
}

func SVGVisit(node SVGNodeInterface, cbk func(node SVGNodeInterface)) {

	children := node.GetChildren()
	for _, child := range children {
		SVGVisit(child, cbk)
	}
	cbk(node)
}

func processGrounds(worldTransform vector.Matrix2, svggrounds []SVGNodeInterface) []mapcontainer.MapGround {

	grounds := make([]mapcontainer.MapGround, 0)

	for _, svgground := range svggrounds {

		subpathes := make([][]PathOperation, 0)
		subpath := make([]PathOperation, 0)
		pathtransform := svgground.GetFullTransform()

		switch typedsvgground := svgground.(type) {
		case *SVGPath:
			{
				svgpath := typedsvgground
				pathoperations := ParseSVGPath(svgpath.GetPath())

				// Split path into subpathes
				// Principle: M => new subpath
				for i, op := range pathoperations {
					if op.Operation == "M" && i > 0 {
						subpathes = append(subpathes, subpath)
						subpath = make([]PathOperation, 0)
					}

					subpath = append(subpath, op)
				}
				break
			}
		case *SVGPolygon:
			{
				svgpolygon := typedsvgground
				subpath = ParseSVGPolygonPoints(svgpolygon.points)
				break
			}
		}

		if len(subpath) > 0 {
			subpathes = append(subpathes, subpath)
		}

		if len(subpathes) == 0 {
			// no path defined on ground !
			continue
		}

		// Make polygons
		ground := mapcontainer.MapGround{
			Id:      svgground.GetId(),
			Outline: make([]mapcontainer.MapPolygon, 0),
		}

		for _, subpath := range subpathes {
			ground.Outline = append(ground.Outline, PathToMapPolygon(
				subpath,
				pathtransform,
				worldTransform,
			))
		}

		///////////////////////////////////////////////////////////////////////
		// Make visible mesh from polygons
		///////////////////////////////////////////////////////////////////////

		contour := make([]*poly2tri.Point, 0)

		for _, point := range ground.Outline[0].Points[:len(ground.Outline[0].Points)-1] {
			contour = append(contour, poly2tri.NewPoint(point.X, point.Y))
		}

		swctx := poly2tri.NewSweepContext(contour, false)

		for _, holePolygon := range ground.Outline[1:] {
			hole := make([]*poly2tri.Point, 0)
			for _, point := range holePolygon.Points[:len(holePolygon.Points)-1] { // avoid last point (repetition of the first point)
				hole = append(hole, poly2tri.NewPoint(point.X, point.Y))
			}

			swctx.AddHole(hole)
		}

		swctx.Triangulate()
		triangles := swctx.GetTriangles()

		// Transform triangles to array of floats ([][][]float64 => []float64)
		positions := make([][][]float64, 0)
		for _, triangle := range triangles {
			positions = append(positions, [][]float64{
				[]float64{triangle.Points[0].GetX(), 0, triangle.Points[0].GetY()},
				[]float64{triangle.Points[1].GetX(), 0, triangle.Points[1].GetY()},
				[]float64{triangle.Points[2].GetX(), 0, triangle.Points[2].GetY()},
			})
		}

		// connect the triangle dots ... counter clockwise
		indices := make([]int, 0)

		for i := 0; i < len(positions); i++ {
			offset := i * 3
			indices = append(indices, offset+0, offset+1, offset+2)
		}

		// flatten positions to vertices array [][][]float64 to []float64
		vertices := make([]float64, 0)
		for i := 0; i < len(positions); i++ {
			for j := 0; j < len(positions[i]); j++ {
				for k := 0; k < len(positions[i][j]); k++ {
					vertices = append(vertices, positions[i][j][k])
				}
			}
		}

		// on recherche les min et max X, Y des coordonnées du mesh

		type AccType struct {
			x float64
			y float64
		}

		meshminmax := func(op int, acc AccType, value float64, i int) AccType {

			if i == 0 || i%3 == 0 {
				// x
				if op < 1 {
					if value < acc.x {
						acc.x = value
					}
				} else if value > acc.x {
					acc.x = value
				}
			} else if i == 2 || (i-2)%3 == 0 {
				// y
				if op < 1 {
					if value < acc.y {
						acc.y = value
					}
				} else if value > acc.y {
					acc.y = value
				}
			}

			return acc
		}

		min := AccType{}
		max := AccType{}

		for i, vertexcoord := range vertices {
			min = meshminmax(-1, min, vertexcoord, i)
			max = meshminmax(+1, max, vertexcoord, i)
		}

		uvs := make([]float64, 0)
		for i, vertexcoord := range vertices {
			if i == 0 || i%3 == 0 {
				// x
				uvs = append(uvs, number.Map(vertexcoord, min.x, max.x, 0, 1))
			} else if i == 2 || (i-2)%3 == 0 {
				// y
				uvs = append(uvs, number.Map(vertexcoord, min.y, max.y, 0, 1))
			}
		}

		ground.Mesh.Vertices = vertices
		ground.Mesh.Uvs = uvs
		ground.Mesh.Indices = indices

		grounds = append(grounds, ground)
	}

	return grounds
}

func processStarts(worldTransform vector.Matrix2, svgstarts []SVGNodeInterface) []mapcontainer.MapStart {

	starts := make([]mapcontainer.MapStart, 0)
	for _, svgstart := range svgstarts {
		switch typednode := svgstart.(type) {
		case *SVGCircle:
			{
				cx, cy := typednode.GetCenter()
				cxt, cyt := typednode.
					GetFullTransform().
					Transform(cx, cy)

				cxt, cyt = worldTransform.Transform(cxt, cyt)

				starts = append(starts, mapcontainer.MapStart{
					Id:    typednode.GetId(),
					Point: mapcontainer.MapPoint{X: cxt, Y: cyt},
				})
			}
		case *SVGEllipse:
			{
				cx, cy := typednode.GetCenter()
				cxt, cyt := typednode.
					GetFullTransform().
					Transform(cx, cy)

				cxt, cyt = worldTransform.Transform(cxt, cyt)

				starts = append(starts, mapcontainer.MapStart{
					Id:    typednode.GetId(),
					Point: mapcontainer.MapPoint{X: cxt, Y: cyt},
				})
			}
		}
	}

	return starts
}

func signum(f float64) int {
	if f < 0 {
		return -1
	}

	return 1
}

func processObjects(worldTransform vector.Matrix2, svgobjects []SVGNodeInterface) ([]mapcontainer.MapPrefabObject, []mapcontainer.MapObstacleObject) {
	prefabObjects := make([]mapcontainer.MapPrefabObject, 0)
	obstacleObjects := make([]mapcontainer.MapObstacleObject, 0)

	circleProcessor := func(node SVGNodeInterface, cx float64, cy float64, radius float64) *mapcontainer.MapPrefabObject {

		ids := GetSVGIDs(node)

		if ids.Contains("ba:object") == -1 {
			return nil
		}

		cxt, cyt := node.
			GetFullTransform().
			Transform(cx, cy)

		cxt, cyt = worldTransform.
			Transform(cxt, cyt)

		radiust, _ := worldTransform.Transform(radius, 0)

		objtype := ""
		if ids.Contains("ba:rocks01") > -1 {
			objtype = "rocks01"
		} else if ids.Contains("ba:rocks02") > -1 {
			objtype = "rocks02"
		} else if ids.Contains("ba:alienBones") > -1 {
			objtype = "alienBones"
		} else if ids.Contains("ba:satellite01") > -1 {
			objtype = "satellite01"
		} else if ids.Contains("ba:satellite02") > -1 {
			objtype = "satellite02"
		} else if ids.Contains("ba:satelliteRand") > -1 {
			if rand.Float64() > 0.5 {
				objtype = "satellite02"
			} else {
				objtype = "satellite01"
			}
		} else if ids.Contains("ba:crater01") > -1 {
			objtype = "crater01"
		} else if ids.Contains("ba:crater02") > -1 {
			objtype = "crater01"
		} else if ids.Contains("ba:craterRand") > -1 {
			if rand.Float64() > 0.5 {
				objtype = "crater02"
			} else {
				objtype = "crater01"
			}

		} else if ids.Contains("ba:rocksRand") > -1 {
			if rand.Float64() > 0.5 {
				objtype = "rocks02"
			} else {
				objtype = "rocks01"
			}

		} else if ids.Contains("ba:station01") > -1 {
			objtype = "station01"
		} else if ids.Contains("ba:station02") > -1 {
			objtype = "station02"
		} else if ids.Contains("ba:stationRand") > -1 {
			if rand.Float64() > 0.5 {
				objtype = "station02"
			} else {
				objtype = "station01"
			}

		} else {
			return nil
		}

		return &mapcontainer.MapPrefabObject{
			Id:       node.GetId(),
			Point:    mapcontainer.MapPoint{X: cxt, Y: cyt},
			Diameter: radiust * 2,
			Type:     objtype,
		}
	}

	applyFunctions := func(functions []SVGIDFunction, obj *mapcontainer.MapPrefabObject) *mapcontainer.MapPrefabObject {

		for _, f := range functions {
			switch f.Function {
			case "randomizeScale":
				{
					args := make([]float64, 2)
					err := json.Unmarshal(f.Args, &args)
					if err != nil {
						utils.Debug("map-builder", "randomizeScale error; "+err.Error())
						os.Exit(1)
					}

					obj.Diameter = number.Map(rand.Float64(), 0, 1, args[0], args[1])
				}
			case "randomizePosition":
				{
					maxdeviation := make([]float64, 1)
					err := json.Unmarshal(f.Args, &maxdeviation)
					if err != nil {
						utils.Debug("map-builder", "randomizePosition error; "+err.Error())
						os.Exit(1)
					}

					devx := number.Map(rand.Float64(), 0, 1, 0, maxdeviation[0]) * float64(signum(rand.Float64()-rand.Float64()))
					devy := number.Map(rand.Float64(), 0, 1, 0, maxdeviation[0]) * float64(signum(rand.Float64()-rand.Float64()))

					obj.Point.X += devx
					obj.Point.Y += devy
				}
			case "randomizeOrientation":
				{
					maxangle := math.Pi * 2
					angle := number.Map(rand.Float64(), 0, 1, 0, maxangle) * float64(signum(rand.Float64()-rand.Float64()))
					obj.Orientation = angle
				}
			}
		}

		return obj
	}

	for _, svgobject := range svgobjects {

		switch typednode := svgobject.(type) {
		case *SVGCircle:
			{
				cx, cy := typednode.GetCenter()
				obj := circleProcessor(typednode, cx, cy, typednode.GetRadius())
				obj = applyFunctions(GetSVGIDs(typednode).GetFunctions(), obj)

				if obj != nil {
					prefabObjects = append(prefabObjects, *obj)
				}
			}
		case *SVGEllipse:
			{
				cx, cy := typednode.GetCenter()
				obj := circleProcessor(typednode, cx, cy, typednode.rx)
				obj = applyFunctions(GetSVGIDs(typednode).GetFunctions(), obj)

				if obj != nil {
					prefabObjects = append(prefabObjects, *obj)
				}
			}
		case *SVGPolygon:
			{
				obstacleObjects = append(obstacleObjects, mapcontainer.MapObstacleObject{
					Id: typednode.GetId(),
					Polygon: PathToMapPolygon(
						ParseSVGPolygonPoints(typednode.points),
						typednode.GetFullTransform(),
						worldTransform,
					),
				})
			}
		}
	}

	return prefabObjects, obstacleObjects
}

func PathToMapPolygon(path []PathOperation, pathtransform vector.Matrix2, worldTransform vector.Matrix2) mapcontainer.MapPolygon {

	newpath := path
	for j, op := range path {
		if strings.ToUpper(op.Operation) == "Z" {
			if newpath[j-1].Coords[0] != newpath[0].Coords[0] || newpath[j-1].Coords[1] != newpath[0].Coords[1] {
				newpath = append(newpath[:j], newpath[0])
			} else {
				newpath = newpath[:j]
			}
		}
	}
	path = newpath

	points := make([]mapcontainer.MapPoint, 0)
	for _, op := range path {
		x, y := pathtransform.
			Transform(op.Coords[0], op.Coords[1])

		x, y = worldTransform.
			Transform(x, y)

		points = append(points, mapcontainer.MapPoint{X: x, Y: y})
	}

	mappoly := mapcontainer.MapPolygon{Points: points}
	polyarr := mappoly.ToVector2Array()

	winding := polygonutils.GetPolygonWindingForCartesianSystem(polyarr)

	// Change polygons winding to CW
	if polygonutils.IsCW(winding) {
		// CW; change to CCW
		log.Println("POLY IS CW; CHANGE WINDING TO CCW")
		newpoly := polygonutils.InvertWinding(polyarr)
		newwinding := polygonutils.GetPolygonWindingForCartesianSystem(newpoly)
		if !polygonutils.IsCCW(newwinding) {
			utils.Debug("map-builder", "Could not change ground polygon winding from CW to CCW")
			os.Exit(1)
		}

		newMapPoint := make([]mapcontainer.MapPoint, len(newpoly))
		for i, point := range newpoly {
			newMapPoint[i] = mapcontainer.MapPoint{
				X: point.GetX(),
				Y: point.GetY(),
			}
		}

		mappoly.Points = newMapPoint
	} else if polygonutils.IsCCW(winding) {
		log.Println("POLY IS CCW; NOTHING TO DO")
	} else {
		utils.Debug("map-builder", "Ground polygon is neither CCW nor CW; something's wrong")
		os.Exit(1)
	}

	// Calculate (outwards pointing) normal for each polygon edge
	normals := make([]mapcontainer.MapPoint, 0)
	polylen := len(polyarr)
	for i := 0; i < polylen; i++ {
		p1 := polyarr[i]
		p2 := polyarr[(i+1)%polylen]
		vec := vector.MakeVector2(p2.GetX(), p2.GetY()).Sub(p1)
		normal := vec.OrthogonalClockwise().Normalize() // clockwise: because poly is CCW, outwards is on the right of the edge
		normals = append(normals, mapcontainer.MapPoint{
			X: normal.GetX(),
			Y: normal.GetY(),
		})
	}

	mappoly.Normals = normals

	return mappoly
}
