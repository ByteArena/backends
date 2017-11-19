package visibility2d

func leftOf(segment *Segment, point Point) bool {
	cross := (segment.PointB.X-segment.PointA.X)*(point.Y-segment.PointA.Y) - (segment.PointB.Y-segment.PointA.Y)*(point.X-segment.PointA.X)
	return cross < 0
}

func interpolate(pointA *EndPoint, pointB *EndPoint, f float64) Point {
	return Point{
		pointA.X*(1-f) + pointB.X*f,
		pointA.Y*(1-f) + pointB.Y*f,
	}
}

func segmentInFrontOf(segmentA, segmentB *Segment, relativePoint Point) bool {
	A1 := leftOf(segmentA, interpolate(segmentB.PointA, segmentB.PointB, 0.01))
	A2 := leftOf(segmentA, interpolate(segmentB.PointB, segmentB.PointA, 0.01))
	A3 := leftOf(segmentA, relativePoint)
	B1 := leftOf(segmentB, interpolate(segmentA.PointA, segmentA.PointB, 0.01))
	B2 := leftOf(segmentB, interpolate(segmentA.PointB, segmentA.PointA, 0.01))
	B3 := leftOf(segmentB, relativePoint)

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

	return false
}
