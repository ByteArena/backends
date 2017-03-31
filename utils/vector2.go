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

func NewVector2(x float64, y float64) *Vector2 {
	return &Vector2{x, y}
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

func (a *Vector2) Clone() *Vector2 {
	return &Vector2{
		x: a.x,
		y: a.y,
	}
}

func (a *Vector2) Add(b *Vector2) *Vector2 {
	a.x += b.x
	a.y += b.y
	return a
}

func (a *Vector2) Sub(b *Vector2) *Vector2 {
	a.x -= b.x
	a.y -= b.y
	return a
}

func (a *Vector2) Scale(scale float64) *Vector2 {
	a.x *= scale
	a.y *= scale
	return a
}

func (a *Vector2) Mult(b *Vector2) *Vector2 {
	a.x *= b.x
	a.y *= b.y
	return a
}

func (a *Vector2) Div(b *Vector2) *Vector2 {
	a.x /= b.x
	a.y /= b.y
	return a
}

// Returns a random unit vector
func RandomVector2() *Vector2 {
	radians := rand.Float64() * math.Pi * 2
	return NewVector2(
		math.Cos(radians),
		math.Sin(radians),
	)
}
