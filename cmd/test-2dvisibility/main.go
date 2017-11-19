package main

type obstacle struct {
	coords [4]float64
	name   string
}

func main() {

	obstacles := []obstacle{
		obstacle{[4]float64{20, 20, 20, 120}, "a"},
		obstacle{[4]float64{20, 20, 100, 20}, "b"},
		obstacle{[4]float64{100, 20, 150, 100}, "c"},
		obstacle{[4]float64{150, 100, 50, 100}, "d"},
		obstacle{[4]float64{0, 50, 200, 50}, "cross"},
	}

	// begin := time.Now()
	// fmt.Println("# 100, 80 ####################################################")
	// spew.Dump(visibility2d.CalculateVisibility(visibility2d.MakePoint(100, 80), obstacles))

	// fmt.Println("# 100, 120 ###################################################")
	// spew.Dump(visibility2d.CalculateVisibility(visibility2d.MakePoint(100, 120), obstacles))

	// fmt.Println("# 300, 300 ###################################################")
	// spew.Dump(visibility2d.CalculateVisibility(visibility2d.MakePoint(300, 300), obstacles))

	// fmt.Println("Took ", float64(time.Now().UnixNano()-begin.UnixNano())/1000000.0, "ms")

	// breakableSegments := make([]breakintersections.ObstacleSegment, len(obstacles))
	// for i := 0; i < len(breakableSegments); i++ {
	// 	v := obstacles[i]
	// 	breakableSegments[i] = breakintersections.ObstacleSegment{
	// 		Points: [2][2]float64{
	// 			[2]float64{v.coords[0], v.coords[1]},
	// 			[2]float64{v.coords[2], v.coords[3]},
	// 		},
	// 		UserData: v.name,
	// 	}
	// }

	// spew.Dump(breakableSegments, breakintersections.BreakIntersections(breakableSegments))

	//spew.Dump(visibility)
}
