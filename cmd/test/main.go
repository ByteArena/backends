package main

import (
	"log"

	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
)

func main() {
	vector.TestSegment2()

	log.Println(trigo.IntersectionWithLineSegment(vector.MakeNullVector2(), vector.MakeVector2(10, 10), vector.MakeVector2(0, 5), vector.MakeVector2(5, 5)))
}
