package vector

import (
	"bytes"
	"fmt"
	"math"

	"github.com/bytearena/bytearena/common/utils/number"
)

type Vector3 struct {
	x float64
	y float64
	z float64
}

func MakeVector3(x float64, y float64, z float64) Vector3 {
	return Vector3{x, y, z}
}

// Returns a null Vector3
func MakeNullVector3() Vector3 {
	return MakeVector3(0, 0, 0)
}

func NewVector3(x float64, y float64, z float64) *Vector3 {
	return &Vector3{x, y, z}
}

func (v Vector3) Get() (float64, float64, float64) {
	return v.x, v.y, v.z
}

func (v Vector3) MarshalJSON() ([]byte, error) {
	propfmt := "%.4f"
	buffer := bytes.NewBufferString("[")
	buffer.WriteString(fmt.Sprintf(propfmt, v.x))
	buffer.WriteString(",")
	buffer.WriteString(fmt.Sprintf(propfmt, v.y))
	buffer.WriteString(",")
	buffer.WriteString(fmt.Sprintf(propfmt, v.z))
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

func (v Vector3) MarshalJSONString() string {
	json, _ := v.MarshalJSON()
	return string(json)
}

func (a Vector3) Clone() Vector3 {
	return Vector3{
		x: a.x,
		y: a.y,
		z: a.z,
	}
}

func (a Vector3) Add(b Vector3) Vector3 {
	a.x += b.x
	a.y += b.y
	a.z += b.z
	return a
}

func (a Vector3) AddScalar(f float64) Vector3 {
	a.x += f
	a.y += f
	a.z += f
	return a
}

func (a Vector3) Sub(b Vector3) Vector3 {
	a.x -= b.x
	a.y -= b.y
	a.z -= b.z
	return a
}

func (a Vector3) SubScalar(f float64) Vector3 {
	a.x -= f
	a.y -= f
	a.z -= f
	return a
}

func (a Vector3) Scale(scale float64) Vector3 {
	a.x *= scale
	a.y *= scale
	a.z *= scale
	return a
}

func (a Vector3) Mult(b Vector3) Vector3 {
	a.x *= b.x
	a.y *= b.y
	a.z *= b.z
	return a
}

func (a Vector3) MultScalar(f float64) Vector3 {
	a.x *= f
	a.y *= f
	a.z *= f
	return a
}

func (a Vector3) Div(b Vector3) Vector3 {
	a.x /= b.x
	a.y /= b.y
	a.z /= b.z
	return a
}

func (a Vector3) DivScalar(f float64) Vector3 {
	a.x /= f
	a.y /= f
	a.z /= f
	return a
}

func (a Vector3) Mag() float64 {
	return math.Sqrt(a.MagSq())
}

func (a Vector3) MagSq() float64 {
	return (a.x*a.x + a.y*a.y + a.z*a.z)
}

func (a Vector3) SetMag(mag float64) Vector3 {
	return a.Normalize().MultScalar(mag)
}

func (a Vector3) Normalize() Vector3 {
	mag := a.Mag()
	if mag > 0 {
		return a.DivScalar(mag)
	}
	return a
}

func (a Vector3) Limit(max float64) Vector3 {

	mSq := a.MagSq()

	if mSq > max*max {
		return a.Normalize().MultScalar(max)
	}

	return a
}

func (a Vector3) Dot(v Vector3) float64 {
	return a.x*v.x + a.y*v.y + a.z*v.z
}

func (a Vector3) IsNull() bool {
	return isZero(a.x) && isZero(a.y) && isZero(a.z)
}

func (a Vector3) Equals(b Vector3) bool {
	return b.Sub(a).IsNull()
}

func (a Vector3) String() string {
	return "<Vector3(" + number.FloatToStr(a.x, 5) + ", " + number.FloatToStr(a.y, 5) + ", " + number.FloatToStr(a.z, 5) + ")>"
}

func (a Vector3) SetAngleOnZAxis(radians float64) Vector3 {
	mag := a.Mag()
	a.x = math.Sin(radians) * mag
	a.y = math.Cos(radians) * mag

	return a
}
