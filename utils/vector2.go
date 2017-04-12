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

func (a Vector2) Limit(max float64) Vector2 {

	mSq := a.MagSq()

	if mSq > max*max {
		return a.Normalize().MultScalar(max)
	}

	return a
}

func (a Vector2) Angle() float64 {
	return math.Atan2(a.y, a.x)
}

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
