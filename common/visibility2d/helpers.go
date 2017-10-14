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
	return []*EndPoint{segment.p1, segment.p2}
}

func calculateEndPointAngles(lightSource Point, segment *Segment) {
	x := lightSource.X
	y := lightSource.Y

	dx := 0.5*(segment.p1.X+segment.p2.X) - x
	dy := 0.5*(segment.p1.Y+segment.p2.Y) - y

	segment.d = (dx * dx) + (dy * dy)
	segment.p1.angle = math.Atan2(segment.p1.Y-y, segment.p1.X-x)
	segment.p2.angle = math.Atan2(segment.p2.Y-y, segment.p2.X-x)
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

func endpointsFromSegments(pov Point, walls []*Segment) []*EndPoint {

	segments := processSegments(pov, walls)

	endpoints := make([]*EndPoint, 0)
	for _, segment := range segments {
		endpoints = append(endpoints, segment.GetEndPoints()...)
	}

	return endpoints
}
