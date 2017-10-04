package main

func lineIntersection(point1, point2, point3, point4 Point) Point {
	s := ((point4.x-point3.x)*(point1.y-point3.y) - (point4.y-point3.y)*(point1.x-point3.x)) /
		((point4.y-point3.y)*(point2.x-point1.x) - (point4.x-point3.x)*(point2.y-point1.y))

	return MakePoint(
		point1.x+s*(point2.x-point1.x),
		point1.y+s*(point2.y-point1.y),
	)
}
