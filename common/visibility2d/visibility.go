package visibility2d

import (
	"math"
	"sort"

	"github.com/bytearena/bytearena/common/utils/vector"
)

func getTrianglePoints(origin Point, angle1, angle2 float64, segment *Segment) [2]Point {

	cosAngle1 := math.Cos(angle1)
	sinAngle1 := math.Sin(angle1)

	cosAngle2 := math.Cos(angle2)
	sinAngle2 := math.Sin(angle2)

	p1 := origin
	p2 := MakePoint(origin.X+cosAngle1, origin.Y+sinAngle1)
	p3 := MakePoint(0, 0)
	p4 := MakePoint(0, 0)

	if segment != nil {
		p3.X = segment.p1.X
		p3.Y = segment.p1.Y
		p4.X = segment.p2.X
		p4.Y = segment.p2.Y
	} else {
		p3.X = origin.X + cosAngle1*200
		p3.Y = origin.Y + sinAngle1*200
		p4.X = origin.X + cosAngle2*200
		p4.Y = origin.Y + sinAngle2*200
	}

	pBegin := lineIntersection(p3, p4, p1, p2)

	p2.X = origin.X + cosAngle2
	p2.Y = origin.Y + sinAngle2

	pEnd := lineIntersection(p3, p4, p1, p2)

	return [2]Point{pBegin, pEnd}
}

type Visible struct {
	Visible  vector.Segment2
	Complete vector.Segment2
	Userdata interface{}
}

func CalculateVisibility(origin Point, segments []*Segment) []Visible {
	openSegments := make([]*Segment, 0)
	output := make([]Visible, 0)
	beginAngle := 0.0

	endpoints := endpointsFromSegments(origin, segments)
	sort.Sort(ByEndpoint(endpoints))

	//spew.Dump(endpoints)

	for pass := 0; pass < 2; pass++ {
		// fmt.Println("## PASS ", pass)
		for i := 0; i < len(endpoints); i++ {
			// fmt.Println("#### point ", i)
			endpoint := endpoints[i]
			var openSegment *Segment
			if len(openSegments) > 0 {
				openSegment = openSegments[0]
			}
			rootchanged := false

			if endpoint.beginsSegment {
				// fmt.Println("####### CHECK A1")
				index := 0
				var segment *Segment
				if len(openSegments) > 0 {
					segment = openSegments[index]
				}
				for segment != nil && segmentInFrontOf(endpoint.segment, segment, origin) {
					index++
					if index < len(openSegments) {
						segment = openSegments[index]
					} else {
						segment = nil
					}
				}

				// fmt.Println("####### CHECK A2", index)

				if segment == nil {
					// fmt.Println("####### CHECK A3")
					openSegments = append(openSegments, endpoint.segment)
					if len(openSegments) == 1 {
						rootchanged = true
						//openSegment = openSegments[0]
					}
				} else {
					// fmt.Println("####### CHECK A4")
					//openSegments.splice(index, 0, endpoint.segment)
					openSegments = append(openSegments, nil)
					copy(openSegments[index+1:], openSegments[index:])
					openSegments[index] = endpoint.segment
					if index == 0 {
						// fmt.Println("####### CHECK A5")
						rootchanged = true
						//openSegment = openSegments[1]
					}
				}
			} else {
				// fmt.Println("####### CHECK B1")
				index := -1
				for j, seg := range openSegments {
					if seg == endpoint.segment {
						index = j
						break
					}
				}

				// fmt.Println("####### CHECK B2", index)

				if index > -1 {
					// fmt.Println("####### CHECK B3")
					//openSegments.splice(index, 1)
					copy(openSegments[index:], openSegments[index+1:])
					openSegments[len(openSegments)-1] = nil
					openSegments = openSegments[:len(openSegments)-1]
					if index == 0 {
						// fmt.Println("####### CHECK B4")
						rootchanged = true
					}
				}
			}

			// fmt.Println("####### CHECK C")

			if rootchanged {
				// fmt.Println("####### CHECK D1")
				if pass == 1 {
					// fmt.Println("####### CHECK D2", origin, beginAngle, endpoint.angle, openSegment.d, openSegment.p1.x, openSegment.p1.y, openSegment.p2.x, openSegment.p2.y)
					if openSegment != nil {
						trianglePoints := getTrianglePoints(origin, beginAngle, endpoint.angle, openSegment)
						output = append(output, Visible{
							Visible: vector.MakeSegment2(
								vector.MakeVector2(trianglePoints[0].X, trianglePoints[0].Y),
								vector.MakeVector2(trianglePoints[1].X, trianglePoints[1].Y),
							),
							Complete: vector.MakeSegment2(
								vector.MakeVector2(openSegment.p1.X, openSegment.p1.Y),
								vector.MakeVector2(openSegment.p2.X, openSegment.p2.Y),
							),
							Userdata: openSegment.userdata,
						})
					}
				}
				// fmt.Println("####### CHECK D3", endpoint.x, endpoint.y, endpoint.segment.d, endpoint.angle)
				beginAngle = endpoint.angle
			}
		}
	}

	// fmt.Println("####### CHECK E")
	// remove visible segments whose length is shorter than a iota

	res := make([]Visible, 0)
	for _, visible := range output {
		if visible.Visible.LengthSq() > 0.001 {
			res = append(res, visible)
		}
	}

	return res
}
