package main

import (
	"fmt"
	"time"

	"github.com/bytearena/bytearena/common/visibility2d"
	"github.com/davecgh/go-spew/spew"
)

func main() {

	obstacles := []*visibility2d.Segment{
		visibility2d.NewSegment(20, 20, 20, 120, "a"),
		visibility2d.NewSegment(20, 20, 100, 20, "b"),
		visibility2d.NewSegment(100, 20, 150, 100, "c"),
		visibility2d.NewSegment(150, 100, 50, 100, "d"),
	}

	pov := visibility2d.MakePoint(300, 300)

	begin := time.Now()
	visibility := visibility2d.CalculateVisibility(pov, obstacles)
	fmt.Println("Took ", float64(time.Now().UnixNano()-begin.UnixNano())/1000000.0, "ms")

	spew.Dump(visibility)
}
