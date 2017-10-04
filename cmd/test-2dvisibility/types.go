package main

type Rectangle struct {
	x      float64
	y      float64
	width  float64
	height float64
}

func MakeRectangle(x, y, width, height float64) Rectangle {
	return Rectangle{
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}
}

type Block Rectangle
type room Rectangle

type Point struct {
	x, y float64
}

func MakePoint(x, y float64) Point {
	return Point{
		x, y,
	}
}

type EndPoint struct {
	Point
	beginsSegment bool
	segment       *Segment
	angle         float64
}

func MakeEndPoint(x, y float64) EndPoint {
	return EndPoint{
		Point:         MakePoint(x, y),
		beginsSegment: false,
		segment:       nil,
		angle:         0,
	}
}

type ByEndpoint []EndPoint

func (coll ByEndpoint) Len() int      { return len(coll) }
func (coll ByEndpoint) Swap(i, j int) { coll[i], coll[j] = coll[j], coll[i] }
func (coll ByEndpoint) Less(i, j int) bool {
	return endpointCompare(coll[i], coll[j]) < 0
}

type Segment struct {
	p1 EndPoint
	p2 EndPoint
	d  float64
}

func NewSegment(x1, y1, x2, y2 float64) *Segment {

	p1 := MakeEndPoint(x1, y1)
	p2 := MakeEndPoint(x2, y2)
	segment := &Segment{
		p1: p1,
		p2: p2,
		d:  0,
	}

	return segment
}
