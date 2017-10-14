package visibility2d

func endpointCompare(pointA, pointB *EndPoint) int {

	if pointA.angle > pointB.angle {
		return 1
	}

	if pointA.angle < pointB.angle {
		return -1
	}

	if !pointA.beginsSegment && pointB.beginsSegment {
		return 1
	}

	if pointA.beginsSegment && !pointB.beginsSegment {
		return -1
	}

	return 0
}
