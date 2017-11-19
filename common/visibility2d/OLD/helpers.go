package visibility2d

import (
	"math"
)

func processSegments(lightSource Point, segments []*Segment) []*Segment {
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		calculateEndPointAngles(lightSource, segment)
		setSegmentBeginning(segment)
	}

	return segments
}

func getSegmentEndPoints(segment *Segment) []*EndPoint {
	return []*EndPoint{segment.PointA, segment.PointB}
}

func calculateEndPointAngles(lightSource Point, segment *Segment) {
	x := lightSource.X
	y := lightSource.Y

	dx := 0.5*(segment.PointA.X+segment.PointB.X) - x
	dy := 0.5*(segment.PointA.Y+segment.PointB.Y) - y

	segment.D = (dx * dx) + (dy * dy)
	segment.PointA.angle = math.Atan2(segment.PointA.Y-y, segment.PointA.X-x)
	segment.PointB.angle = math.Atan2(segment.PointB.Y-y, segment.PointB.X-x)
}

func setSegmentBeginning(segment *Segment) {
	dAngle := segment.PointB.angle - segment.PointA.angle

	if dAngle <= -math.Pi {
		dAngle += 2 * math.Pi
	}

	if dAngle > math.Pi {
		dAngle -= 2 * math.Pi
	}

	segment.PointA.beginsSegment = dAngle > 0
	segment.PointB.beginsSegment = !segment.PointA.beginsSegment
}

func endpointsFromSegments(pov Point, walls []*Segment) []*EndPoint {

	segments := processSegments(pov, walls)

	endpoints := make([]*EndPoint, 0)
	for _, segment := range segments {
		endpoints = append(endpoints, segment.GetEndPoints()...)
	}

	return endpoints
}
