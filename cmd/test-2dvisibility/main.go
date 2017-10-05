package main

import (
	"fmt"
	"time"

	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/davecgh/go-spew/spew"
)

func main() {

	room := MakeRectangle(0, 0, 700, 500)

	walls := []*Segment{
		NewSegment(20, 20, 20, 120),
		NewSegment(20, 20, 100, 20),
		NewSegment(100, 20, 150, 100),
		NewSegment(150, 100, 50, 100),
	}

	blocks := []Rectangle{
		MakeRectangle(50, 150, 20, 20),
		MakeRectangle(150, 150, 40, 80),
		MakeRectangle(400, 400, 40, 40),
	}

	lightSource := MakePoint(300, 300)

	begin := time.Now()
	endpoints := loadMap(room, blocks, walls, lightSource)
	var visibility []vector.Segment2
	for k := 0; k < 1000; k++ {
		visibility = calculateVisibility(lightSource, endpoints)
	}

	fmt.Println("Took ", float64(time.Now().UnixNano()-begin.UnixNano())/1000000.0, "ms")
	spew.Dump(visibility)
}
