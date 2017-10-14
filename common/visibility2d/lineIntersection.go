package visibility2d

func lineIntersection(point1, point2, point3, point4 Point) Point {
	s := ((point4.X-point3.X)*(point1.Y-point3.Y) - (point4.Y-point3.Y)*(point1.X-point3.X)) /
		((point4.Y-point3.Y)*(point2.X-point1.X) - (point4.X-point3.X)*(point2.Y-point1.Y))

	return MakePoint(
		point1.X+s*(point2.X-point1.X),
		point1.Y+s*(point2.Y-point1.Y),
	)
}
