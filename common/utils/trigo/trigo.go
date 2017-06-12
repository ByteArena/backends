package trigo

import (
	"math"

	"github.com/bytearena/bytearena/common/utils/number"
	"github.com/bytearena/bytearena/common/utils/vector"
)

func IntersectionWithLineSegment(p vector.Vector2, p2 vector.Vector2, q vector.Vector2, q2 vector.Vector2) (intersection vector.Vector2, intersects bool, colinear bool, parallel bool) {

	r := p2.Sub(p)
	s := q2.Sub(q)
	rxs := r.Cross(s)
	qpxr := q.Sub(p).Cross(r)

	// If r x s = 0 and (q - p) x r = 0, then the two lines are collinear.
	if number.IsZero(rxs) && number.IsZero(qpxr) {
		// 1. If either  0 <= (q - p) * r <= r * r or 0 <= (p - q) * s <= * s
		// then the two lines are overlapping,
		qSubPTimesR := q.Sub(p).Dot(r)
		pSubQTimesS := p.Sub(q).Dot(s)
		rSquared := r.Dot(r)
		sSquared := s.Dot(s)

		if (qSubPTimesR >= 0 && qSubPTimesR <= rSquared) || (pSubQTimesS >= 0 && pSubQTimesS <= sSquared) {
			return vector.MakeNullVector2(), true, true, true
		}

		// 2. If neither 0 <= (q - p) * r = r * r nor 0 <= (p - q) * s <= s * s
		// then the two lines are collinear but disjoint.
		// No need to implement this expression, as it follows from the expression above.
		return vector.MakeNullVector2(), false, true, true
	}

	// 3. If r x s = 0 and (q - p) x r != 0, then the two lines are parallel and non-intersecting.
	if number.IsZero(rxs) && !number.IsZero(qpxr) {
		return vector.MakeNullVector2(), false, false, true
	}

	t := q.Sub(p).Cross(s) / rxs
	u := q.Sub(p).Cross(r) / rxs

	// 4. If r x s != 0 and 0 <= t <= 1 and 0 <= u <= 1
	// the two line segments meet at the point p + t r = q + u s.
	if !number.IsZero(rxs) && (0 <= t && t <= 1) && (0 <= u && u <= 1) {
		// We can calculate the intersection point using either t or u.
		return p.Add(r.MultScalar(t)), true, false, false
	}

	// 5. Otherwise, the two line segments are not parallel but do not intersect.
	return vector.MakeNullVector2(), false, false, true
}

func IntersectionWithLineSegmentCheckOnly(p1 vector.Vector2, p2 vector.Vector2, p3 vector.Vector2, p4 vector.Vector2) (intersect bool) {
	a := p2.Sub(p1)
	b := p3.Sub(p4)
	c := p1.Sub(p3)

	ax, ay := a.Get()
	bx, by := b.Get()
	cx, cy := c.Get()

	alphaNumerator := by*cx - bx*cy
	alphaDenominator := ay*bx - ax*by
	betaNumerator := ax*cy - ay*cx
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

func LinesIntersectionPoint(p0 vector.Vector2, p1 vector.Vector2, p2 vector.Vector2, p3 vector.Vector2) (point vector.Vector2, parallel bool) {

	p0x, p0y := p0.Get()
	p1x, p1y := p1.Get()
	p2x, p2y := p2.Get()
	p3x, p3y := p3.Get()

	s1_x := p1x - p0x
	s1_y := p1y - p0y
	s2_x := p3x - p2x
	s2_y := p3y - p2y

	s := (-s1_y*(p0x-p2x) + s1_x*(p0y-p2y)) / (-s2_x*s1_y + s1_x*s2_y)
	t := (s2_x*(p0y-p2y) - s2_y*(p0x-p2x)) / (-s2_x*s1_y + s1_x*s2_y)

	if s >= 0 && s <= 1 && t >= 0 && t <= 1 {
		// Collision detected
		return vector.MakeVector2(p0x+(t*s1_x), p0y+(t*s1_y)), false
	}

	// No collision
	return vector.MakeNullVector2(), true
}

// http://devmag.org.za/2009/04/17/basic-collision-detection-in-2d-part-2/
func LineCircleIntersectionPoints(LineP1 vector.Vector2, LineP2 vector.Vector2, CircleCentre vector.Vector2, Radius float64) []vector.Vector2 {

	LocalP1 := LineP1.Sub(CircleCentre)
	LocalP2 := LineP2.Sub(CircleCentre)
	// Precalculate this value. We use it often
	P2MinusP1 := LocalP2.Sub(LocalP1)

	p2minusp1x, p2minusp1y := P2MinusP1.Get()
	localp1x, localp1y := LocalP1.Get()

	a := P2MinusP1.MagSq()
	b := 2 * ((p2minusp1x * localp1x) + (p2minusp1y * localp1y))
	c := LocalP1.MagSq() - (Radius * Radius)

	delta := b*b - (4 * a * c)
	if delta < 0 {
		// No intersection
		return make([]vector.Vector2, 0)
	}

	if delta == 0 {
		u := -b / (2.0 * a)

		// Use LineP1 instead of LocalP1 because we want our answer in global space, not the circle's local space
		res := make([]vector.Vector2, 1)
		res[0] = LineP1.Add(P2MinusP1.MultScalar(u))
		return res
	}

	// (delta > 0) // Two intersections
	SquareRootDelta := math.Sqrt(delta)

	u1 := (-b + SquareRootDelta) / (2 * a)
	u2 := (-b - SquareRootDelta) / (2 * a)

	res := make([]vector.Vector2, 2)
	res[0] = LineP1.Add(P2MinusP1.MultScalar(u1))
	res[1] = LineP1.Add(P2MinusP1.MultScalar(u2))

	return res
}

func PointOnLineSegment(p vector.Vector2, a vector.Vector2, b vector.Vector2) bool {
	t := 0.0001

	px, py := p.Get()
	ax, ay := a.Get()
	bx, by := b.Get()

	// ensure points are collinear
	zero := (bx-ax)*(py-ay) - (px-ax)*(by-ay)
	if zero > t || zero < -t {
		return false
	}

	// check if x-coordinates are not equal
	if ax-bx > t || bx-ax > t {
		// ensure x is between a.x & b.x (use tolerance)
		if ax > bx {
			return px+t > bx && px-t < ax
		} else {
			return px+t > ax && px-t < bx
		}
	}

	// ensure y is between a.y & b.y (use tolerance)
	if ay > by {
		return py+t > by && py-t < ay
	}

	return py+t > ay && py-t < by
}

func FullCircleAngleToSignedHalfCircleAngle(rad float64) float64 {
	if rad > math.Pi { // 180° en radians
		rad -= math.Pi * 2 // 360° en radian
	} else if rad < -math.Pi {
		rad += math.Pi * 2 // 360° en radian
	}

	return rad
}
