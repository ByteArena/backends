package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/vector"
	poly2tri "github.com/netgusto/poly2tri-go"
)

func main() {
	source := flag.String("in", "", "Input svg file; required")
	pxperunit := flag.Float64("pxperunit", 10.0, "Number of svg px per map unit; default 10.0")
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
	bjsonmap, _ := json.MarshalIndent(builtmap, "", "  ")

	fmt.Println(string(bjsonmap))
}

func buildMap(svg SVGNode, pxperunit float64) mapcontainer.MapContainer {
	// Path + Fill=groundcolor: ground
	// Circle/Ellipse + Fill=red: starting spot
	// Circle/Ellipse + Fill=obstaclecolor: normalized obstacle
	// Polygon + Fill=obstaclecolor: custom obstacle

	worldTransform := vector.MakeMatrix2()
	if pxperunit != 1.0 {
		worldTransform = worldTransform.Scale(1/pxperunit, 1/pxperunit)
	}

	groundcolor := "#E3A478"
	startcolor := "#D0011B"
	obstaclecolor := "#8B572A"

	svggrounds := make([]SVGNode, 0)
	svgstarts := make([]SVGNode, 0)
	svgobstacles := make([]SVGNode, 0)
	svgcustomobstacles := make([]SVGNode, 0)

	SVGVisit(svg, func(node SVGNode) {
		switch /*typednode :=*/ node.(type) {
		case *SVGPath:
			{
				if node.GetFill() == groundcolor {
					svggrounds = append(svggrounds, node)
				}
			}
		case *SVGCircle:
			{
				fill := node.GetFill()
				if fill == startcolor {
					svgstarts = append(svgstarts, node)
				} else if fill == obstaclecolor {
					svgobstacles = append(svgobstacles, node)
				}
			}
		case *SVGEllipse:
			{
				fill := node.GetFill()
				if fill == startcolor {
					svgstarts = append(svgstarts, node)
				} else if fill == obstaclecolor {
					svgobstacles = append(svgobstacles, node)
				}
			}
		case *SVGPolygon:
			{
				if node.GetFill() == obstaclecolor {
					svgcustomobstacles = append(svgcustomobstacles, node)
				}
			}
		}
	})

	/************************************/
	/* Processing grounds */
	/************************************/

	grounds := make([]mapcontainer.MapGround, 0)

	for _, svgground := range svggrounds {
		svgpath := svgground.(*SVGPath)
		pathtransform := svgpath.GetFullTransform()
		pathoperations := ParseSVGPath(svgpath.GetPath())

		// Split path into subpathes
		// Principle: M => new subpath
		subpathes := make([][]PathOperation, 0)
		subpath := make([]PathOperation, 0)
		for i, op := range pathoperations {
			if op.Operation == "M" && i > 0 {
				subpathes = append(subpathes, subpath)
				subpath = make([]PathOperation, 0)
			}

			subpath = append(subpath, op)
		}

		subpathes = append(subpathes, subpath)

		// Normalize coords for each subpath
		// Z => expand
		// m => M (relative => abs)

		for i, subpath := range subpathes {
			for j, op := range subpath {
				if strings.ToUpper(op.Operation) == "Z" {
					// expand Z into L X Y

					firstop := subpath[0]
					x := firstop.Coords[0]
					y := firstop.Coords[1]

					subpath[j] = PathOperation{
						Operation: "L",
						Coords:    []float64{x, y},
					}
				}
			}

			subpathes[i] = subpath
		}

		// Make polygons
		ground := mapcontainer.MapGround{Id: svgpath.GetId(), Outline: make([]mapcontainer.MapPolygon, 0)}
		for _, subpath := range subpathes {
			points := make([]mapcontainer.MapPoint, 0)
			for _, op := range subpath {
				x, y := pathtransform.
					Mul(worldTransform).
					Transform(op.Coords[0], op.Coords[1])

				points = append(points, mapcontainer.MapPoint{x, y})
			}

			ground.Outline = append(ground.Outline, mapcontainer.MapPolygon{Points: points})
		}

		// Make meshes from polygons

		contour := make([]*poly2tri.Point, 0)

		for _, point := range ground.Outline[0].Points[:len(ground.Outline[0].Points)-2] { // avoid last point (repetition of the first point)
			contour = append(contour, poly2tri.NewPoint(point.X, point.Y))
		}

		swctx := poly2tri.NewSweepContext(contour, false)

		for _, holePolygon := range ground.Outline[1:] {
			hole := make([]*poly2tri.Point, 0)
			for _, point := range holePolygon.Points[:len(holePolygon.Points)-2] { // avoid last point (repetition of the first point)
				hole = append(hole, poly2tri.NewPoint(point.X, point.Y))
			}

			swctx.AddHole(hole)
		}

		swctx.Triangulate()
		triangles := swctx.GetTriangles()
		// log.Println(triangles)
		// panic("toooo")

		// jsonmesh, _ := json.MarshalIndent(triangles, "", "  ")
		// fmt.Println(string(jsonmesh))

		// fmt.Println("<svg height=\"1000\" width=\"1000\">")
		// for _, triangle := range triangles {
		// 	fmt.Print(fmt.Sprintf("	<polygon points=\"%f,%f %f,%f %f,%f\" style=\"fill:lime;stroke:purple;stroke-width:1\" />\n",
		// 		triangle.Points[0].GetX()*10, triangle.Points[0].GetY()*10,
		// 		triangle.Points[1].GetX()*10, triangle.Points[1].GetY()*10,
		// 		triangle.Points[2].GetX()*10, triangle.Points[2].GetY()*10,
		// 	))
		// }
		// fmt.Println("</svg>")

		ground.Mesh = triangles

		grounds = append(grounds, ground)
	}

	/************************************/
	/* Processing STARTS */
	/************************************/

	starts := make([]mapcontainer.MapStart, 0)
	for _, svgstart := range svgstarts {
		switch typednode := svgstart.(type) {
		case *SVGCircle:
			{
				cx, cy := typednode.GetCenter()
				cxt, cyt := typednode.
					GetFullTransform().
					Mul(worldTransform).
					Transform(cx, cy)

				starts = append(starts, mapcontainer.MapStart{
					Id:    "id",
					Point: mapcontainer.MapPoint{cxt, cyt},
				})
			}
		case *SVGEllipse:
			{
				cx, cy := typednode.GetCenter()
				cxt, cyt := typednode.
					GetFullTransform().
					Mul(worldTransform).
					Transform(cx, cy)

				starts = append(starts, mapcontainer.MapStart{
					Id:    "id",
					Point: mapcontainer.MapPoint{cxt, cyt},
				})
			}
		}
	}

	/************************************/
	/* TODO: Processing OBSTACLES */
	/************************************/

	/************************************/
	/* TODO: Processing CUSTOMOBSTACLES */
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

	return builtmap
}

func SVGVisit(node SVGNode, cbk func(node SVGNode)) {

	children := node.GetChildren()
	for _, child := range children {
		SVGVisit(child, cbk)
	}
	cbk(node)
}