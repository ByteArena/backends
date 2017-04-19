package utils

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
)

type Vector2 struct {
	x float64
	y float64
}

func MakeVector2(x float64, y float64) Vector2 {
	return Vector2{x, y}
}

func NewVector2(x float64, y float64) *Vector2 {
	return &Vector2{x, y}
}

func (v Vector2) Get() (float64, float64) {
	return v.x, v.y
}

func (v Vector2) MarshalJSON() ([]byte, error) {
	propfmt := "%.4f"
	buffer := bytes.NewBufferString("[")
	buffer.WriteString(fmt.Sprintf(propfmt, v.x))
	buffer.WriteString(",")
	buffer.WriteString(fmt.Sprintf(propfmt, v.y))
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

func (a Vector2) Clone() Vector2 {
	return Vector2{
		x: a.x,
		y: a.y,
	}
}

func (a Vector2) Add(b Vector2) Vector2 {
	a.x += b.x
	a.y += b.y
	return a
}

func (a Vector2) AddScalar(f float64) Vector2 {
	a.x += f
	a.y += f
	return a
}

func (a Vector2) Sub(b Vector2) Vector2 {
	a.x -= b.x
	a.y -= b.y
	return a
}

func (a Vector2) SubScalar(f float64) Vector2 {
	a.x -= f
	a.y -= f
	return a
}

func (a Vector2) Scale(scale float64) Vector2 {
	a.x *= scale
	a.y *= scale
	return a
}

func (a Vector2) Mult(b Vector2) Vector2 {
	a.x *= b.x
	a.y *= b.y
	return a
}

func (a Vector2) MultScalar(f float64) Vector2 {
	a.x *= f
	a.y *= f
	return a
}

func (a Vector2) Div(b Vector2) Vector2 {
	a.x /= b.x
	a.y /= b.y
	return a
}

func (a Vector2) DivScalar(f float64) Vector2 {
	a.x /= f
	a.y /= f
	return a
}

func (a Vector2) Mag() float64 {
	return math.Sqrt(a.MagSq())
}

func (a Vector2) MagSq() float64 {
	return (a.x*a.x + a.y*a.y)
}

func (a Vector2) SetMag(mag float64) Vector2 {
	return a.Normalize().MultScalar(mag)
}

func (a Vector2) Normalize() Vector2 {
	mag := a.Mag()
	if mag > 0 {
		return a.DivScalar(mag)
	}
	return a
}

func (a Vector2) SetAngle(radians float64) Vector2 {
	mag := a.Mag()
	a.x = math.Sin(radians) * mag
	a.y = math.Cos(radians) * mag

	return a
}

func (a Vector2) Limit(max float64) Vector2 {

	mSq := a.MagSq()

	if mSq > max*max {
		return a.Normalize().MultScalar(max)
	}

	return a
}

func (a Vector2) Angle() float64 {
	if a.x == 0 && a.y == 0 {
		return 0
	}

	angle := math.Atan2(a.y, a.x)

	// Quart de tour à gauche
	angle = math.Pi/2.0 - angle

	if angle < 0 {
		angle += 2 * math.Pi
	}

	return angle
}

func (a Vector2) Cross(v Vector2) float64 {
	return a.x*v.y - a.y*v.x
}

func (a Vector2) Dot(v Vector2) float64 {
	return a.x*v.x - a.y*v.y
}

var epsilon float64 = 0.0000000001

func (a Vector2) IsZero() bool {
	return math.Abs(a.x) < epsilon && math.Abs(a.y) < epsilon
}

func IsZero(f float64) bool {
	return math.Abs(f) < epsilon
}

func (a Vector2) Equals(b Vector2) bool {
	return b.Sub(a).IsZero()
}

/// <summary>
/// Test whether two line segments intersect. If so, calculate the intersection point.
/// <see cref="http://stackoverflow.com/a/14143738/292237"/>
/// </summary>
/// <param name="p">Vector to the start point of p.</param>
/// <param name="p2">Vector to the end point of p.</param>
/// <param name="q">Vector to the start point of q.</param>
/// <param name="q2">Vector to the end point of q.</param>
/// <param name="intersection">The point of intersection, if any.</param>
/// <param name="considerOverlapAsIntersect">Do we consider overlapping lines as intersecting?
/// </param>
/// <returns>True if an intersection point was found.</returns>

func intersectionWithLineSegment(p Vector2, p2 Vector2, q Vector2, q2 Vector2) (intersection Vector2, intersects bool, colinear bool, parallel bool) {

	r := p2.Sub(p)
	s := q2.Sub(q)
	rxs := r.Cross(s)
	qpxr := q.Sub(p).Cross(r)

	// If r x s = 0 and (q - p) x r = 0, then the two lines are collinear.
	if IsZero(rxs) && IsZero(qpxr) {
		// 1. If either  0 <= (q - p) * r <= r * r or 0 <= (p - q) * s <= * s
		// then the two lines are overlapping,
		qSubPTimesR := q.Sub(p).Dot(r)
		pSubQTimesS := p.Sub(q).Dot(s)
		rSquared := r.Dot(r)
		sSquared := s.Dot(s)

		if (qSubPTimesR >= 0 && qSubPTimesR <= rSquared) || (pSubQTimesS >= 0 && pSubQTimesS <= sSquared) {
			return MakeVector2(0, 0), true, true, true
		}

		// 2. If neither 0 <= (q - p) * r = r * r nor 0 <= (p - q) * s <= s * s
		// then the two lines are collinear but disjoint.
		// No need to implement this expression, as it follows from the expression above.
		return MakeVector2(0, 0), false, true, true
	}

	// 3. If r x s = 0 and (q - p) x r != 0, then the two lines are parallel and non-intersecting.
	if IsZero(rxs) && !IsZero(qpxr) {
		return MakeVector2(0, 0), false, false, true
	}

	t := q.Sub(p).Cross(s) / rxs
	u := q.Sub(p).Cross(r) / rxs

	// 4. If r x s != 0 and 0 <= t <= 1 and 0 <= u <= 1
	// the two line segments meet at the point p + t r = q + u s.
	if !IsZero(rxs) && (0 <= t && t <= 1) && (0 <= u && u <= 1) {
		// We can calculate the intersection point using either t or u.
		return p.Add(r.MultScalar(t)), true, false, false
	}

	// 5. Otherwise, the two line segments are not parallel but do not intersect.
	return MakeVector2(0, 0), false, false, false
}

func (a Vector2) IntersectionWithLineSegment(q Vector2, q2 Vector2) (intersection Vector2, intersects bool, colinear bool, parallel bool) {
	// return intersectionWithLineSegment(MakeVector2(0, 0), a, q, q2)
	vec, intersects, parallel := a.IntersectionWithLineSegmentOld(q, q2)
	return vec, intersects, false, parallel
}

func IntersectionWithLineSegmentCheckOnly(p1 Vector2, p2 Vector2, p3 Vector2, p4 Vector2) (intersect bool) {
	a := p2.Sub(p1)
	b := p3.Sub(p4)
	c := p1.Sub(p3)

	alphaNumerator := b.y*c.x - b.x*c.y
	alphaDenominator := a.y*b.x - a.x*b.y
	betaNumerator := a.x*c.y - a.y*c.x
	betaDenominator := alphaDenominator

	doIntersect := true

	if alphaDenominator == 0 || betaDenominator == 0 {
		doIntersect = false
	} else {
		if alphaDenominator > 0 {
			if alphaNumerator < 0 || alphaNumerator > alphaDenominator {
				doIntersect = false
			}
		} else if alphaNumerator > 0 || alphaNumerator < alphaDenominator {
			doIntersect = false
		}

		if doIntersect && betaDenominator > 0 {
			if betaNumerator < 0 || betaNumerator > betaDenominator {
				doIntersect = false
			}
		} else if betaNumerator > 0 || betaNumerator < betaDenominator {
			doIntersect = false
		}
	}

	return doIntersect
}

func (relvec Vector2) IntersectionWithLineSegmentOld(segstart Vector2, segend Vector2) (point Vector2, intersects bool, parallel bool) {
	vecpointstart := MakeVector2(0, 0) // because vec is relative
	vecpointend := relvec
	intersectionpoint, parallel := LineIntersectionPointBis(
		vecpointstart,
		vecpointend,
		segstart,
		segend,
	)

	if parallel {
		return intersectionpoint, false, true
	}

	// Have to determine if intersection point is on segment and on given vector
	// => Point has to be both on vector and on line segment
	//
	// Point on vector ? Dist from vectorstart <= Mag(vec)
	// Point on line segment ? Dist from segstart <= Mag(segment)

	relvecmagsq := vecpointend.Sub(vecpointstart).MagSq()
	distfromvecstartsq := intersectionpoint.Sub(vecpointstart).MagSq()
	if distfromvecstartsq > relvecmagsq {
		return intersectionpoint, false, false
	}

	segmagsq := segend.Sub(segstart).MagSq()
	distfromsegstartsq := intersectionpoint.Sub(segstart).MagSq()
	if distfromsegstartsq > segmagsq {
		return intersectionpoint, false, false
	}

	return intersectionpoint, true, false
}

//function getLineIntersection(p0_x, p0_y, p1_x, p1_y, p2_x, p2_y, p3_x, p3_y) {
func LineIntersectionPointBis(p0 Vector2, p1 Vector2, p2 Vector2, p3 Vector2) (point Vector2, parallel bool) {

	//var s1_x = p1_x - p0_x;
	s1_x := p1.x - p0.x
	//var s1_y = p1_y - p0_y;
	s1_y := p1.y - p0.y
	//var s2_x = p3_x - p2_x;
	s2_x := p3.x - p2.x
	//var s2_y = p3_y - p2_y;
	s2_y := p3.y - p2.y

	//var s = (-s1_y * (p0_x - p2_x) + s1_x * (p0_y - p2_y)) / (-s2_x * s1_y + s1_x * s2_y);
	s := (-s1_y*(p0.x-p2.x) + s1_x*(p0.y-p2.y)) / (-s2_x*s1_y + s1_x*s2_y)
	//var t = (s2_x * (p0_y - p2_y) - s2_y * (p0_x - p2_x)) / (-s2_x * s1_y + s1_x * s2_y);
	t := (s2_x*(p0.y-p2.y) - s2_y*(p0.x-p2.x)) / (-s2_x*s1_y + s1_x*s2_y)

	if s >= 0 && s <= 1 && t >= 0 && t <= 1 {
		// Collision detected
		//return [p0_x + (t * s1_x), p0_y + (t * s1_y)];
		return MakeVector2(p0.x+(t*s1_x), p0.y+(t*s1_y)), false
	}

	// No collision
	return MakeVector2(0, 0), true
}

// http://devmag.org.za/2009/04/17/basic-collision-detection-in-2d-part-2/
func CircleLineCollision(LineP1 Vector2, LineP2 Vector2, CircleCentre Vector2, Radius float64) []Vector2 {

	LocalP1 := LineP1.Sub(CircleCentre)
	LocalP2 := LineP2.Sub(CircleCentre)
	// Precalculate this value. We use it often
	P2MinusP1 := LocalP2.Sub(LocalP1)

	a := P2MinusP1.MagSq()
	b := 2 * ((P2MinusP1.x * LocalP1.x) + (P2MinusP1.y * LocalP1.y))
	c := LocalP1.MagSq() - (Radius * Radius)

	delta := b*b - (4 * a * c)
	if delta < 0 {
		// No intersection
		return make([]Vector2, 0)
	}

	if delta == 0 {
		u := -b / (2.0 * a)

		// Use LineP1 instead of LocalP1 because we want our answer in global space, not the circle's local space
		res := make([]Vector2, 1)
		res[0] = LineP1.Add(P2MinusP1.MultScalar(u))
		return res
	}

	// (delta > 0) // Two intersections
	SquareRootDelta := math.Sqrt(delta)

	u1 := (-b + SquareRootDelta) / (2 * a)
	u2 := (-b - SquareRootDelta) / (2 * a)

	res := make([]Vector2, 2)
	res[0] = LineP1.Add(P2MinusP1.MultScalar(u1))
	res[1] = LineP1.Add(P2MinusP1.MultScalar(u2))

	return res
}

func (p Vector2) PointOnLineSegment(a Vector2, b Vector2) bool {
	t := 0.0001

	// ensure points are collinear
	zero := (b.x-a.x)*(p.y-a.y) - (p.x-a.x)*(b.y-a.y)
	if zero > t || zero < -t {
		return false
	}

	// check if x-coordinates are not equal
	if a.x-b.x > t || b.x-a.x > t {
		// ensure x is between a.x & b.x (use tolerance)
		if a.x > b.x {
			return p.x+t > b.x && p.x-t < a.x
		} else {
			return p.x+t > a.x && p.x-t < b.x
		}
	}

	// ensure y is between a.y & b.y (use tolerance)
	if a.y > b.y {
		return p.y+t > b.y && p.y-t < a.y
	}

	return p.y+t > a.y && p.y-t < b.y
}

/*
public static bool PointOnLine2D (this Vector2 p, Vector2 a, Vector2 b, float t = 1E-03f)
{
    // ensure points are collinear
    var zero = (b.x - a.x) * (p.y - a.y) - (p.x - a.x) * (b.y - a.y);
    if (zero > t || zero < -t) return false;

    // check if x-coordinates are not equal
    if (a.x - b.x > t || b.x - a.x > t)
        // ensure x is between a.x & b.x (use tolerance)
        return a.x > b.x
            ? p.x + t > b.x && p.x - t < a.x
            : p.x + t > a.x && p.x - t < b.x;

    // ensure y is between a.y & b.y (use tolerance)
    return a.y > b.y
        ? p.y + t > b.y && p.y - t < a.y
        : p.y + t > a.y && p.y - t < b.y;
}
*/

func (a Vector2) ToArray() []float64 {
	res := make([]float64, 2)
	res[0] = a.x
	res[1] = a.y
	return res
}

func (a Vector2) String() string {
	return "<Vector2(" + FloatToStr(a.x, 5) + ", " + FloatToStr(a.y, 5) + ")>"
}

// Returns a random unit vector
func MakeRandomVector2() Vector2 {
	radians := rand.Float64() * math.Pi * 2
	return MakeVector2(
		math.Cos(radians),
		math.Sin(radians),
	)
}
