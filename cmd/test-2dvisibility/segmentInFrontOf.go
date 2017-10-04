package main

func leftOf(segment *Segment, point Point) bool {
	cross := (segment.p2.x-segment.p1.x)*(point.y-segment.p1.y) - (segment.p2.y-segment.p1.y)*(point.x-segment.p1.x)
	return cross < 0
}

func interpolate(pointA EndPoint, pointB EndPoint, f float64) Point {
	return Point{
		pointA.x*(1-f) + pointB.x*f,
		pointA.y*(1-f) + pointB.y*f,
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
