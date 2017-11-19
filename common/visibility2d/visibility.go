package visibility2d

import (
	"math"
	"sort"

	"github.com/bytearena/bytearena/common/types/datastructures"
)

type Point struct {
	X, Y float64
}

type EndPoint struct {
	Point
	Begin     bool
	Segment   *Segment
	Angle     float64
	Visualize bool
}

type Segment struct {
	P1       *EndPoint
	P2       *EndPoint
	D        float64
	UserData interface{}
}

type VisibleSegment struct {
	P1              Point
	P2              Point
	CompleteSegment *Segment
}

type Visibility struct {
	Segments              []*Segment
	EndPoints             []*EndPoint
	Center                Point
	Open                  datastructures.DLL
	Output                []VisibleSegment
	IntersectionsDetected [][]Point
}

func MakeVisibility() Visibility {
	return Visibility{
		Segments:              make([]*Segment, 0),
		EndPoints:             make([]*EndPoint, 0),
		Open:                  datastructures.DLL{},
		Center:                Point{0, 0},
		Output:                make([]VisibleSegment, 0),
		IntersectionsDetected: make([][]Point, 0),
	}
}

// func (visibility *Visibility) loadEdgeOfMap(size, margin float64) {
// 	visibility.addSegment(margin, margin, margin, size-margin)
// 	visibility.addSegment(margin, size-margin, size-margin, size-margin)
// 	visibility.addSegment(size-margin, size-margin, size-margin, margin)
// 	visibility.addSegment(size-margin, margin, margin, margin)
// }

// Load a set of square blocks, plus any other line segments
// func (visibility *Visibility) loadMap(size, margin float64, walls []Segment) {

// 	visibility.Segments = make([]*Segment, 0)
// 	visibility.EndPoints = make([]EndPoint, 0)
// 	visibility.loadEdgeOfMap(size, margin)

// 	for _, wall := range walls {
// 		visibility.addSegment(wall.P1.X, wall.P1.Y, wall.P2.X, wall.P2.Y)
// 	}
// }

func (visibility *Visibility) AddSegment(x1, y1, x2, y2 float64, userdata interface{}) {

	segment := &Segment{}

	p1 := &EndPoint{
		Point:     Point{x1, y1},
		Segment:   segment,
		Visualize: true,
	}

	p2 := &EndPoint{
		Point:     Point{x2, y2},
		Segment:   segment,
		Visualize: false,
	}

	segment.P1 = p1
	segment.P2 = p2
	segment.D = 0.0
	segment.UserData = userdata

	visibility.Segments = append(visibility.Segments, segment)
	visibility.EndPoints = append(visibility.EndPoints, p1, p2)

}

func (visibility *Visibility) SetLightLocation(x, y float64) {
	visibility.Center.X = x
	visibility.Center.Y = y

	for _, segment := range visibility.Segments {

		dx := 0.5*(segment.P1.X+segment.P2.X) - x
		dy := 0.5*(segment.P1.Y+segment.P2.Y) - y
		// NOTE: we only use this for comparison so we can use
		// distance squared instead of distance. However in
		// practice the sqrt is plenty fast and this doesn't
		// really help in this situation.
		segment.D = dx*dx + dy*dy

		// NOTE: future optimization: we could record the quadrant
		// and the y/x or x/y ratio, and sort by (quadrant,
		// ratio), instead of calling atan2. See
		// <https://github.com/mikolalysenko/compare-slope> for a
		// library that does this. Alternatively, calculate the
		// angles and use bucket sort to get an O(N) sort.
		segment.P1.Angle = math.Atan2(segment.P1.Y-y, segment.P1.X-x)
		segment.P2.Angle = math.Atan2(segment.P2.Y-y, segment.P2.X-x)

		dAngle := segment.P2.Angle - segment.P1.Angle
		if dAngle <= -math.Pi {
			dAngle += 2 * math.Pi
		}
		if dAngle > math.Pi {
			dAngle -= 2 * math.Pi
		}
		segment.P1.Begin = (dAngle > 0.0)
		segment.P2.Begin = !segment.P1.Begin
	}
}

type byAngle []*EndPoint

func (coll byAngle) Len() int      { return len(coll) }
func (coll byAngle) Swap(i, j int) { coll[i], coll[j] = coll[j], coll[i] }
func (coll byAngle) Less(i, j int) bool {

	a := coll[i]
	b := coll[j]

	// Traverse in angle order
	if a.Angle > b.Angle {
		return false
	}

	if a.Angle < b.Angle {
		return true
	}

	// But for ties (common), we want Begin nodes before End nodes
	if !a.Begin && b.Begin {
		return false
	}

	if a.Begin && !b.Begin {
		return true
	}

	return false
}

func leftOf(s *Segment, p Point) bool {
	// This is based on a 3d cross product, but we don't need to
	// use z coordinate inputs (they're 0), and we only need the
	// sign. If you're annoyed that cross product is only defined
	// in 3d, see "outer product" in Geometric Algebra.
	// <http://en.wikipedia.org/wiki/Geometric_algebra>
	cross := (s.P2.X-s.P1.X)*(p.Y-s.P1.Y) - (s.P2.Y-s.P1.Y)*(p.X-s.P1.X)
	return cross < 0
	// Also note that this is the naive version of the test and
	// isn't numerically robust. See
	// <https://github.com/mikolalysenko/robust-arithmetic> for a
	// demo of how this fails when a point is very close to the
	// line.
}

func interpolate(p, q Point, f float64) Point {
	return Point{p.X*(1-f) + q.X*f, p.Y*(1-f) + q.Y*f}
}

// Helper: do we know that segment a is in front of b?
// Implementation not anti-symmetric (that is to say,
// _segment_in_front_of(a, b) != (!_segment_in_front_of(b, a)).
// Also note that it only has to work in a restricted set of cases
// in the visibility algorithm; I don't think it handles all
// cases. See http://www.redblobgames.com/articles/visibility/segment-sorting.html
func (visibility *Visibility) _segment_in_front_of(a, b *Segment, relativeTo Point) bool {
	// NOTE: we slightly shorten the segments so that
	// intersections of the endpoints (common) don't count as
	// intersections in this algorithm
	var A1 = leftOf(a, interpolate(b.P1.Point, b.P2.Point, 0.01))
	var A2 = leftOf(a, interpolate(b.P2.Point, b.P1.Point, 0.01))
	var A3 = leftOf(a, relativeTo)
	var B1 = leftOf(b, interpolate(a.P1.Point, a.P2.Point, 0.01))
	var B2 = leftOf(b, interpolate(a.P2.Point, a.P1.Point, 0.01))
	var B3 = leftOf(b, relativeTo)

	// NOTE: this algorithm is probably worthy of a short article
	// but for now, draw it on paper to see how it works. Consider
	// the line A1-A2. If both B1 and B2 are on one side and
	// relativeTo is on the other side, then A is in between the
	// viewer and B. We can do the same with B1-B2: if A1 and A2
	// are on one side, and relativeTo is on the other side, then
	// B is in between the viewer and A.
	if B1 == B2 && B2 != B3 {
		return true
	}
	if A1 == A2 && A2 == A3 {
		return true
	}
	if A1 == A2 && A2 != A3 {
		return false
	}
	if B1 == B2 && B2 == B3 {
		return false
	}

	// If A1 != A2 and B1 != B2 then we have an intersection.
	// Expose it for the GUI to show a message. A more robust
	// implementation would split segments at intersections so
	// that part of the segment is in front and part is behind.
	visibility.IntersectionsDetected = append(
		visibility.IntersectionsDetected,
		[]Point{a.P1.Point, a.P2.Point, b.P1.Point, b.P2.Point},
	)

	return false

	// NOTE: previous implementation was a.d < b.d. That's simpler
	// but trouble when the segments are of dissimilar sizes. If
	// you're on a grid and the segments are similarly sized, then
	// using distance will be a simpler and faster implementation.
}

// Run the algorithm, sweeping over all or part of the circle to find
// the visible area, represented as a set of triangles
func (visibility *Visibility) Sweep() {
	maxAngle := 999.0

	visibility.Output = make([]VisibleSegment, 0) // output set of triangles
	visibility.IntersectionsDetected = make([][]Point, 0)

	sort.Sort(byAngle(visibility.EndPoints))

	visibility.Open.Clear()
	var beginAngle = 0.0

	// At the beginning of the sweep we want to know which
	// segments are active. The simplest way to do this is to make
	// a pass collecting the segments, and make another pass to
	// both collect and process them. However it would be more
	// efficient to go through all the segments, figure out which
	// ones intersect the initial sweep line, and then sort them.
	for pass := 0; pass < 2; pass++ {
		for _, p := range visibility.EndPoints {
			if pass == 1 && p.Angle > maxAngle {
				// Early exit for the visualization to show the sweep process
				break
			}

			var current_old *Segment = nil
			if !visibility.Open.Empty() {
				current_old = visibility.Open.Head.Val.(*Segment)
			}

			if p.Begin {

				// Insert into the right place in the list
				node := visibility.Open.Head

				for node != nil {
					valAsSegment := node.Val.(*Segment)

					if !visibility._segment_in_front_of(p.Segment, valAsSegment, visibility.Center) {
						break
					}

					node = node.Next
				}

				if node == nil {
					visibility.Open.Append(p.Segment)
				} else {
					visibility.Open.InsertBefore(node, p.Segment)
				}
			} else {
				visibility.Open.RemoveVal(p.Segment)
			}

			var current_new *Segment = nil
			if !visibility.Open.Empty() {
				current_new = visibility.Open.Head.Val.(*Segment)
			}

			//log.Println(pass, current_old, current_new)

			if current_old != current_new {
				if pass == 1 {
					visibility.addTriangle(beginAngle, p.Angle, current_old)
				}
				beginAngle = p.Angle
			}
		}
	}
}

func lineIntersection(p1, p2, p3, p4 Point) Point {
	// From http://paulbourke.net/geometry/lineline2d/
	var s = ((p4.X-p3.X)*(p1.Y-p3.Y) - (p4.Y-p3.Y)*(p1.X-p3.X)) / ((p4.Y-p3.Y)*(p2.X-p1.X) - (p4.X-p3.X)*(p2.Y-p1.Y))
	return Point{p1.X + s*(p2.X-p1.X), p1.Y + s*(p2.Y-p1.Y)}
}

func (visibility *Visibility) addTriangle(angle1, angle2 float64, segment *Segment) {

	if segment == nil {
		return
	}

	var p1 Point = visibility.Center
	var p2 Point = Point{visibility.Center.X + math.Cos(angle1), visibility.Center.Y + math.Sin(angle1)}
	var p3 Point = Point{0.0, 0.0}
	var p4 Point = Point{0.0, 0.0}

	// Stop the triangle at the intersecting segment
	p3.X = segment.P1.X
	p3.Y = segment.P1.Y
	p4.X = segment.P2.X
	p4.Y = segment.P2.Y

	var pBegin = lineIntersection(p3, p4, p1, p2)

	p2.X = visibility.Center.X + math.Cos(angle2)
	p2.Y = visibility.Center.Y + math.Sin(angle2)
	var pEnd = lineIntersection(p3, p4, p1, p2)

	visibility.Output = append(visibility.Output, VisibleSegment{
		P1:              pBegin,
		P2:              pEnd,
		CompleteSegment: segment,
	})
}
