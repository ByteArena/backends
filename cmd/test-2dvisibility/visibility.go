package main

import (
	"math"
	"sort"

	"github.com/bytearena/bytearena/common/utils/vector"
)

func getTrianglePoints(origin Point, angle1, angle2 float64, segment *Segment) [2]Point {

	p1 := origin
	p2 := MakePoint(origin.x+math.Cos(angle1), origin.y+math.Sin(angle1))
	p3 := MakePoint(0, 0)
	p4 := MakePoint(0, 0)

	if segment != nil {
		p3.x = segment.p1.x
		p3.y = segment.p1.y
		p4.x = segment.p2.x
		p4.y = segment.p2.y
	} else {
		p3.x = origin.x + math.Cos(angle1)*200
		p3.y = origin.y + math.Sin(angle1)*200
		p4.x = origin.x + math.Cos(angle2)*200
		p4.y = origin.y + math.Sin(angle2)*200
	}

	pBegin := lineIntersection(p3, p4, p1, p2)

	p2.x = origin.x + math.Cos(angle2)
	p2.y = origin.y + math.Sin(angle2)

	pEnd := lineIntersection(p3, p4, p1, p2)

	return [2]Point{pBegin, pEnd}
}

func calculateVisibility(origin Point, endpoints []*EndPoint) []vector.Segment2 {
	openSegments := make([]*Segment, 0)
	output := make([][2]Point, 0)
	beginAngle := 0.0

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
					trianglePoints := getTrianglePoints(origin, beginAngle, endpoint.angle, openSegment)
					output = append(output, trianglePoints)
				}
				// fmt.Println("####### CHECK D3", endpoint.x, endpoint.y, endpoint.segment.d, endpoint.angle)
				beginAngle = endpoint.angle
			}
		}
	}

	// fmt.Println("####### CHECK E")
	// remove visible segments whose length is shorter than a iota

	res := make([]vector.Segment2, 0)
	for _, points := range output {
		seg2 := vector.MakeSegment2(
			vector.MakeVector2(points[0].x, points[0].y),
			vector.MakeVector2(points[1].x, points[1].y),
		)

		if seg2.LengthSq() > 0.001 {
			res = append(res, seg2)
		}
	}

	return res
}
