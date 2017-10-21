package visibility2d

func leftOf(segment *Segment, point Point) bool {
	cross := (segment.p2.X-segment.p1.X)*(point.Y-segment.p1.Y) - (segment.p2.Y-segment.p1.Y)*(point.X-segment.p1.X)
	return cross < 0
}

func interpolate(pointA *EndPoint, pointB *EndPoint, f float64) Point {
	return Point{
		pointA.X*(1-f) + pointB.X*f,
		pointA.Y*(1-f) + pointB.Y*f,
	}
}

func segmentInFrontOf(segmentA, segmentB *Segment, relativePoint Point) bool {
	A1 := leftOf(segmentA, interpolate(segmentB.p1, segmentB.p2, 0.01))
	A2 := leftOf(segmentA, interpolate(segmentB.p2, segmentB.p1, 0.01))
	A3 := leftOf(segmentA, relativePoint)
	B1 := leftOf(segmentB, interpolate(segmentA.p1, segmentA.p2, 0.01))
	B2 := leftOf(segmentB, interpolate(segmentA.p2, segmentA.p1, 0.01))
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
