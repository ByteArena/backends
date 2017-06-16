package vector

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"

	"github.com/bytearena/bytearena/common/utils/number"
)

type Vector2 struct {
	x float64
	y float64
}

func MakeVector2(x float64, y float64) Vector2 {
	return Vector2{x, y}
}

// Returns a random unit vector
func MakeRandomVector2() Vector2 {
	radians := rand.Float64() * math.Pi * 2
	return MakeVector2(
		math.Cos(radians),
		math.Sin(radians),
	)
}

// Returns a null vector2
func MakeNullVector2() Vector2 {
	return MakeVector2(0, 0)
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

func (v Vector2) MarshalJSONString() string {
	json, _ := v.MarshalJSON()
	return string(json)
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

func (a Vector2) OrthogonalClockwise() Vector2 {
	return MakeVector2(a.y, -a.x)
}

func (a Vector2) OrthogonalCounterClockwise() Vector2 {
	return MakeVector2(-a.y, a.x)
}

func (a Vector2) Center() Vector2 {
	return a.MultScalar(0.5)
}

func (a Vector2) Translate(translation Vector2) Vector2 {
	return a.Add(translation)
}

func (a Vector2) MoveCenterTo(newcenterpos Vector2) Vector2 {
	return a.Translate(newcenterpos.Sub(a.Center()))
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
	return a.x*v.x + a.y*v.y
}

func (a Vector2) IsNull() bool {
	return isZero(a.x) && isZero(a.y)
}

func (a Vector2) Equals(b Vector2) bool {
	return b.Sub(a).IsNull()
}

func (a Vector2) String() string {
	return "<Vector2(" + number.FloatToStr(a.x, 5) + ", " + number.FloatToStr(a.y, 5) + ")>"
}

var epsilon float64 = 0.000001

func isZero(f float64) bool {
	return math.Abs(f) < epsilon
}
