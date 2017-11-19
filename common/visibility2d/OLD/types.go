package visibility2d

type Point struct {
	X, Y float64
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

func NewEndPoint(x, y float64) *EndPoint {
	return &EndPoint{
		Point:         MakePoint(x, y),
		beginsSegment: false,
		segment:       nil,
		angle:         0,
	}
}

type ByEndpoint []*EndPoint

func (coll ByEndpoint) Len() int      { return len(coll) }
func (coll ByEndpoint) Swap(i, j int) { coll[i], coll[j] = coll[j], coll[i] }
func (coll ByEndpoint) Less(i, j int) bool {
	return endpointCompare(coll[i], coll[j]) < 0
}

type Segment struct {
	PointA   *EndPoint
	PointB   *EndPoint
	D        float64
	Userdata interface{}
}

func NewSegment(x1, y1, x2, y2 float64, userdata interface{}) *Segment {

	p1 := NewEndPoint(x1, y1)
	p2 := NewEndPoint(x2, y2)
	segment := &Segment{
		PointA:   p1,
		PointB:   p2,
		D:        0,
		Userdata: userdata,
	}

	p1.segment = segment
	p2.segment = segment

	return segment
}

func (s Segment) GetEndPoints() []*EndPoint {
	return []*EndPoint{s.PointA, s.PointB}
}
