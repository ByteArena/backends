package main

import "math"

func calculateEndPointAngles(lightSource Point, segment *Segment) {
	x := lightSource.x
	y := lightSource.y

	dx := 0.5*(segment.p1.x+segment.p2.x) - x
	dy := 0.5*(segment.p1.y+segment.p2.y) - y

	segment.d = (dx * dx) + (dy * dy)
	segment.p1.angle = math.Atan2(segment.p1.y-y, segment.p1.x-x)
	segment.p2.angle = math.Atan2(segment.p2.y-y, segment.p2.x-x)
}

func setSegmentBeginning(segment *Segment) {
	dAngle := segment.p2.angle - segment.p1.angle

	if dAngle <= -math.Pi {
		dAngle += 2 * math.Pi
	}

	if dAngle > math.Pi {
		dAngle -= 2 * math.Pi
	}

	segment.p1.beginsSegment = dAngle > 0
	segment.p2.beginsSegment = !segment.p1.beginsSegment
}

func processSegments(lightSource Point, segments []*Segment) []*Segment {
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		calculateEndPointAngles(lightSource, segment)
		setSegmentBeginning(segment)
	}

	return segments
}

func getSegmentEndPoints(segment *Segment) []*EndPoint {
	return []*EndPoint{segment.p1, segment.p2}
}

///////////////////////////////////////////////////////////////////////////////

func loadMap(room Rectangle, blocks []Rectangle, walls []*Segment, lightSource Point) []*EndPoint {

	blocksegments := make([]*Segment, 0)
	for _, block := range blocks {
		blocksegments = append(blocksegments, rectangleToSegments(block)...)
	}

	segments := make([]*Segment, 0)
	segments = append(segments, rectangleToSegments(room)...)
	segments = append(segments, blocksegments...)
	segments = append(segments, walls...)

	segments = processSegments(lightSource, segments)

	endpoints := make([]*EndPoint, 0)
	for _, segment := range segments {
		endpoints = append(endpoints, getSegmentEndPoints(segment)...)
	}

	return endpoints
}

type Corners struct {
	nw [2]float64
	sw [2]float64
	ne [2]float64
	se [2]float64
}

func getCorners(x, y, width, height float64) Corners {
	return Corners{
		nw: [2]float64{x, y},
		sw: [2]float64{x, y + height},
		ne: [2]float64{x + width, y},
		se: [2]float64{x + width, y + height},
	}
}

func segmentsFromCorners(corners Corners) []*Segment {
	return []*Segment{
		NewSegment(corners.nw[0], corners.nw[1], corners.ne[0], corners.ne[1]),
		NewSegment(corners.nw[0], corners.nw[1], corners.sw[0], corners.sw[1]),
		NewSegment(corners.ne[0], corners.ne[1], corners.se[0], corners.se[1]),
		NewSegment(corners.sw[0], corners.sw[1], corners.se[0], corners.se[1]),
	}
}

func rectangleToSegments(rectangle Rectangle) []*Segment {
	return segmentsFromCorners(getCorners(rectangle.x, rectangle.y, rectangle.width, rectangle.height))
}
