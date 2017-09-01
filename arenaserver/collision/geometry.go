package collision

import (
	"errors"

	polyclip "github.com/akavel/polyclip-go"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
)

func makePoly(centerA, centerB vector.Vector2, radiusA, radiusB float64) []vector.Vector2 {
	hasMoved := !centerA.Equals(centerB)

	if !hasMoved {
		return nil
	}

	AB := vector.MakeSegment2(centerA, centerB)

	// on détermine les 4 points formant le rectangle orienté définissant la trajectoire de l'object en movement

	polyASide := AB.OrthogonalToACentered().SetLengthFromCenter(radiusA * 2) // si AB vertical, A à gauche, B à droite
	polyBSide := AB.OrthogonalToBCentered().SetLengthFromCenter(radiusB * 2) // si AB vertical, A à gauche, B à droite

	/*

		B2*--------------------*A2
		  |                    |
		B *                    * A
		  |                    |
		B1*--------------------*A1

	*/

	polyA1, polyA2 := polyASide.Get()
	polyB1, polyB2 := polyBSide.Get()

	return []vector.Vector2{polyA1, polyA2, polyB2, polyB1}
}

func clipOrientedRectangles(polyOne, polyTwo []vector.Vector2) []vector.Vector2 {

	contourOne := make(polyclip.Contour, len(polyOne))
	for i, p := range polyOne {
		contourOne[i] = polyclip.Point{X: p.GetX(), Y: p.GetY()}
	}

	contourTwo := make(polyclip.Contour, len(polyTwo))
	for i, p := range polyTwo {
		contourTwo[i] = polyclip.Point{X: p.GetX(), Y: p.GetY()}
	}

	subject := polyclip.Polygon{contourOne}
	clipping := polyclip.Polygon{contourTwo}

	result := subject.Construct(polyclip.INTERSECTION, clipping)
	if len(result) == 0 || len(result[0]) < 3 {
		return nil
	}

	res := make([]vector.Vector2, len(result[0]))
	for i, p := range result[0] {
		res[i] = vector.MakeVector2(p.X, p.Y)
	}

	return res
}

func getGeometryObjectBoundingBox(position vector.Vector2, radius float64) (bottomLeft vector.Vector2, topRight vector.Vector2) {
	x, y := position.Get()
	return vector.MakeVector2(x-radius, y-radius), vector.MakeVector2(x+radius, y+radius)
}

func GetTrajectoryBoundingBox(beginPoint vector.Vector2, beginRadius float64, endPoint vector.Vector2, endRadius float64) (*rtreego.Rect, error) {
	beginBottomLeft, beginTopRight := getGeometryObjectBoundingBox(beginPoint, beginRadius)
	endBottomLeft, endTopRight := getGeometryObjectBoundingBox(endPoint, endRadius)

	bbTopLeft, bbDimensions := state.GetBoundingBox([]vector.Vector2{beginBottomLeft, beginTopRight, endBottomLeft, endTopRight})

	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

	bbRegion, err := rtreego.NewRect(bbTopLeft, bbDimensions)
	if err != nil {
		return nil, errors.New("Error in getTrajectoryBoundingBox: could not define bbRegion in rTree")
	}

	// fmt.Println("----------------------------------------------------------------")
	// show.Dump(bbTopLeft, bbDimensions, bbRegion)
	// fmt.Println("----------------------------------------------------------------")

	return bbRegion, nil
}

func isInsideGroundSurface(mapmemoization *state.MapMemoization, point vector.Vector2) bool {

	px, py := point.Get()

	bb, _ := rtreego.NewRect([]float64{px - 0.005, py - 0.005}, []float64{0.01, 0.01})
	matchingTriangles := mapmemoization.RtreeSurface.SearchIntersect(bb)

	if len(matchingTriangles) == 0 {
		return false
	}

	// On vérifie que le point est bien dans un des triangles
	for _, spatial := range matchingTriangles {
		triangle := spatial.(*state.TriangleRtreeWrapper)
		if trigo.PointIsInTriangle(point, triangle.Points[0], triangle.Points[1], triangle.Points[2]) {
			return true
		}
	}

	return false
}

func isInsideCollisionMesh(mapmemoization *state.MapMemoization, point vector.Vector2) bool {

	px, py := point.Get()

	bb, _ := rtreego.NewRect([]float64{px - 0.005, py - 0.005}, []float64{0.01, 0.01})
	matchingTriangles := mapmemoization.RtreeCollisions.SearchIntersect(bb)

	if len(matchingTriangles) == 0 {
		return false
	}

	// On vérifie que le point est bien dans un des triangles
	for _, spatial := range matchingTriangles {
		triangle := spatial.(*state.TriangleRtreeWrapper)
		if trigo.PointIsInTriangle(point, triangle.Points[0], triangle.Points[1], triangle.Points[2]) {
			return true
		}
	}

	return false
}
