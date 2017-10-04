package main

import (
	"math"
	"sort"
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

func calculateVisibility(origin Point, endpoints []EndPoint) [][2]Point {
	openSegments := make([]*Segment, 0)
	output := make([][2]Point, 0)
	beginAngle := 0.0

	sort.Sort(ByEndpoint(endpoints))

	for pass := 0; pass < 2; pass++ {
		for i := 0; i < len(endpoints); i++ {
			endpoint := endpoints[i]
			openSegment := openSegments[0]

			if endpoint.beginsSegment {
				index := 0
				segment := openSegments[index]
				for segment != nil && segmentInFrontOf(endpoint.segment, segment, origin) {
					index++
					segment = openSegments[index]
				}

				if segment == nil {
					openSegments = append(openSegments, endpoint.segment)
				} else {
					//openSegments.splice(index, 0, endpoint.segment)
					openSegments = append(openSegments, nil)
					copy(openSegments[index+1:], openSegments[index:])
					openSegments[index] = endpoint.segment
				}
			} else {
				index := -1
				for j, seg := range openSegments {
					if seg == endpoint.segment {
						index = j
						break
					}
				}

				if index > -1 {
					copy(openSegments[index:], openSegments[index+1:])
					openSegments[len(openSegments)-1] = nil
					openSegments = openSegments[:len(openSegments)-1]
					//openSegments.splice(index, 1)
				}
			}

			if openSegment != openSegments[0] {
				if pass == 1 {
					trianglePoints := getTrianglePoints(origin, beginAngle, endpoint.angle, openSegment)
					output = append(output, trianglePoints)
				}
				beginAngle = endpoint.angle
			}
		}
	}

	return output
}
