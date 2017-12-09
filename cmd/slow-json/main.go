package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bytearena/core/common/utils"
)

type vec struct {
	X, Y float64
}

type vecWithMarshaler struct {
	X, Y float64
}

var floatformat = byte('f')

var b = []byte{'[', '2', '5', '.', '1', '2', '3', '4', ',', '3', '3', '.', '1', '2', '3', ']'}

func (v vecWithMarshaler) MarshalJSON() ([]byte, error) {
	//return json.Marshal([]float64{v.X, v.Y})
	b := []byte{'['}
	b = strconv.AppendFloat(b, v.X, floatformat, 4, 64)
	b = append(b, byte(','))
	b = strconv.AppendFloat(b, v.Y, floatformat, 4, 64)
	return append(b, byte(']')), nil
}

func main() {
	capacity := 10000

	res := make([]vec, capacity)

	for i := 0; i < capacity; i++ {
		res[i].X, res[i].Y = 25.1234, 33.123
	}

	watch := utils.MakeStopwatch("json-serialization")
	watch.Start("without-marshaler")
	x, _ := json.Marshal(res)
	watch.Stop("without-marshaler")

	fmt.Println(watch.String(), len(x))

	///////////////////////////////////////////////////////////////

	resWithMarshaler := make([]vecWithMarshaler, capacity)

	for i := 0; i < capacity; i++ {
		resWithMarshaler[i].X, resWithMarshaler[i].Y = 25.1234, 33.123
	}

	watch.Start("with-marshaler")
	x2, _ := json.Marshal(resWithMarshaler)
	watch.Stop("with-marshaler")

	fmt.Println(watch.String(), len(x), len(x2))

	///////////////////////////////////////////////////////////////

	resArrayOfFloats := make([][2]float64, capacity)

	for i := 0; i < capacity; i++ {
		resArrayOfFloats[i][0], resArrayOfFloats[i][1] = 25.1234, 33.123
	}

	watch.Start("with-array-of-floats")
	x3, _ := json.Marshal(resArrayOfFloats)
	watch.Stop("with-array-of-floats")

	fmt.Println(watch.String(), len(x3), len(x3))
}
