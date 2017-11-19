package breakintersections

import (
	"math"

	"github.com/bytearena/bytearena/common/visibility2d"
)

const epsilon = 0.1

type Segment struct {
	Points   [2][2]float64
	UserData interface{}
}

func distance(a, b [2]float64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	return dx*dx + dy*dy
}

func equal(a, b [2]float64) bool {
	return math.Abs(a[0]-b[0]) < epsilon && math.Abs(a[1]-b[1]) < epsilon
}

func intersectLines(a1, a2, b1, b2 [2]float64) (p [2]float64, intersects bool) {
	var dbx = b2[0] - b1[0]
	var dby = b2[1] - b1[1]
	var dax = a2[0] - a1[0]
	var day = a2[1] - a1[1]

	var uB = dby*dax - dbx*day
	if uB != 0 {
		var ua = (dbx*(a1[1]-b1[1]) - dby*(a1[0]-b1[0])) / uB
		return [2]float64{a1[0] - ua*-dax, a1[1] - ua*-day}, true
	}

	return [2]float64{}, false
}

func isOnSegment(xi, yi, xj, yj, xk, yk float64) bool {
	return (xi <= xk || xj <= xk) && (xk <= xi || xk <= xj) &&
		(yi <= yk || yj <= yk) && (yk <= yi || yk <= yj)
}

func computeDirection(xi, yi, xj, yj, xk, yk float64) int {
	var a = (xk - xi) * (yj - yi)
	var b = (xj - xi) * (yk - yi)
	if a < b {
		return -1
	}

	if a > b {
		return 1
	}

	return 0
}

func doLineSegmentsIntersect(x1, y1, x2, y2, x3, y3, x4, y4 float64) bool {
	var d1 = computeDirection(x3, y3, x4, y4, x1, y1)
	var d2 = computeDirection(x3, y3, x4, y4, x2, y2)
	var d3 = computeDirection(x1, y1, x2, y2, x3, y3)
	var d4 = computeDirection(x1, y1, x2, y2, x4, y4)
	return (((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0))) ||
		(d1 == 0 && isOnSegment(x3, y3, x4, y4, x1, y1)) ||
		(d2 == 0 && isOnSegment(x3, y3, x4, y4, x2, y2)) ||
		(d3 == 0 && isOnSegment(x1, y1, x2, y2, x3, y3)) ||
		(d4 == 0 && isOnSegment(x1, y1, x2, y2, x4, y4))
}

func BreakIntersections(segments []Segment) []Segment {
	var output = make([]Segment, 0)

	for i := 0; i < len(segments); i++ {
		intersections := make([][2]float64, 0)

		for j := 0; j < len(segments); j++ {

			if i == j {
				continue
			}

			if doLineSegmentsIntersect(segments[i].Points[0][0], segments[i].Points[0][1], segments[i].Points[1][0], segments[i].Points[1][1], segments[j].Points[0][0], segments[j].Points[0][1], segments[j].Points[1][0], segments[j].Points[1][1]) {

				if intersectPoint, intersects := intersectLines(segments[i].Points[0], segments[i].Points[1], segments[j].Points[0], segments[j].Points[1]); intersects {
					if equal(intersectPoint, segments[i].Points[0]) || equal(intersectPoint, segments[i].Points[1]) {
						continue
					}

					intersections = append(intersections, intersectPoint)
				}
			}
		}

		start := [2]float64{segments[i].Points[0][0], segments[i].Points[0][1]}

		for len(intersections) > 0 {

			var endIndex = 0
			var endDis = distance(start, intersections[0])

			for j := 1; j < len(intersections); j++ {
				var dis = distance(start, intersections[j])
				if dis < endDis {
					endDis = dis
					endIndex = j
				}
			}

			output = append(output, Segment{
				Points: [2][2]float64{
					[2]float64{start[0], start[1]},
					[2]float64{intersections[endIndex][0], intersections[endIndex][1]},
				},
				UserData: segments[i].UserData,
			})
			start[0] = intersections[endIndex][0]
			start[1] = intersections[endIndex][1]
			intersections = append(intersections[:endIndex], intersections[endIndex+1:]...)
		}

		output = append(output, Segment{
			Points: [2][2]float64{
				start,
				[2]float64{segments[i].Points[1][0], segments[i].Points[1][1]},
			},
			UserData: segments[i].UserData,
		})
	}
	return output
}

func OnlyVisible(position [2]float64, perceptionitems []Segment) []Segment {

	scaledUpSegments := make([]Segment, 0)
	for _, item := range perceptionitems {
		scaledUpSegments = append(scaledUpSegments, Segment{
			Points: [2][2]float64{
				[2]float64{item.Points[0][0] * 1000.0, item.Points[0][1] * 1000.0},
				[2]float64{item.Points[1][0] * 1000.0, item.Points[1][1] * 1000.0},
			},
			UserData: item.UserData,
		})
	}

	scaledUpSegments = BreakIntersections(scaledUpSegments)

	//return perceptionitems

	visibleSegments := make([]Segment, 0)

	visibility := visibility2d.MakeVisibility()

	for _, item := range scaledUpSegments {
		visibility.AddSegment(
			item.Points[0][0], item.Points[0][1],
			item.Points[1][0], item.Points[1][1],
			item.UserData,
		)
	}

	visibility.SetLightLocation(position[0], position[1])

	visibility.Sweep()
	for _, visibleSegment := range visibility.Output {
		visibleSegments = append(visibleSegments, Segment{
			Points: [2][2]float64{
				[2]float64{visibleSegment.P1.X / 1000.0, visibleSegment.P1.Y / 1000.0},
				[2]float64{visibleSegment.P2.X / 1000.0, visibleSegment.P2.Y / 1000.0},
			},
			UserData: visibleSegment.CompleteSegment.UserData,
		})
	}

	//res := compute(position, perceptionitems)
	//spew.Dump(res)
	return visibleSegments
}

/*

type intstack []int

func (stack intstack) pop() int {
	if stack.len() == 0 {
		panic("cannot pop empty stack")
	}

	res := stack[stack.len()-1]
	stack = stack[:stack.len()-2]

	return res
}

func (stack intstack) len() int {
	return len(stack)
}

func (stack intstack) peek(pos int) int {
	if stack.len() <= pos {
		return -1
	}

	return stack[pos]
}

func compute(position [2]float64, segments []Segment) [][2]float64 {
	bounded := make([]Segment, 0)
	minX := position[0]
	minY := position[1]
	maxX := position[0]
	maxY := position[1]

	for i := 0; i < len(segments); i++ {

		minX = math.Min(minX, segments[i].Points[0][0])
		minY = math.Min(minY, segments[i].Points[0][1])
		maxX = math.Max(maxX, segments[i].Points[0][0])
		maxY = math.Max(maxY, segments[i].Points[0][1])

		minX = math.Min(minX, segments[i].Points[1][0])
		minY = math.Min(minY, segments[i].Points[1][1])
		maxX = math.Max(maxX, segments[i].Points[1][0])
		maxY = math.Max(maxY, segments[i].Points[1][1])

		bounded = append(bounded, Segment{
			Points: [2][2]float64{
				[2]float64{segments[i].Points[0][0], segments[i].Points[0][1]},
				[2]float64{segments[i].Points[1][0], segments[i].Points[1][1]},
			},
			UserData: segments[i].UserData,
		})
	}

	minX--
	minY--
	maxX++
	maxY++

	// world boundaries
	bounded = append(bounded, Segment{
		Points: [2][2]float64{
			[2]float64{minX, minY},
			[2]float64{maxX, minY},
		},
		UserData: nil,
	})

	bounded = append(bounded, Segment{
		Points: [2][2]float64{
			[2]float64{maxX, minY},
			[2]float64{maxX, maxY},
		},
		UserData: nil,
	})

	bounded = append(bounded, Segment{
		Points: [2][2]float64{
			[2]float64{maxX, maxY},
			[2]float64{minX, maxY},
		},
		UserData: nil,
	})

	bounded = append(bounded, Segment{
		Points: [2][2]float64{
			[2]float64{minX, maxY},
			[2]float64{minX, minY},
		},
		UserData: nil,
	})

	polygon := make([][2]float64, 0)

	sorted := sortPoints(position, bounded)

	themap := make([]int, len(bounded))

	for i := 0; i < len(themap); i++ {
		themap[i] = -1
	}

	heap := make(intstack, 0)

	start := [2]float64{position[0] + 1, position[1]}

	for i := 0; i < len(bounded); i++ {

		a1 := angle(bounded[i].Points[0], position)
		a2 := angle(bounded[i].Points[1], position)

		active := false

		if a1 > -180 && a1 <= 0 && a2 <= 180 && a2 >= 0 && a2-a1 > 180 {
			active = true
		}

		if a2 > -180 && a2 <= 0 && a1 <= 180 && a1 >= 0 && a1-a2 > 180 {
			active = true
		}

		if active {
			insert(i, heap, position, bounded, start, themap)
		}
	}

	for i := 0; i < len(sorted); {
		extend := false
		shorten := false
		orig := i
		vertex := bounded[sorted[i].zero].Points[sorted[i].one]
		var old_segment int = -1
		old_segment = heap.peek(0)

		for {
			if themap[sorted[i].zero] != -1 {
				if sorted[i].zero == old_segment {
					extend = true
					vertex = bounded[sorted[i].zero].Points[sorted[i].one]
				}
				remove(themap[sorted[i].zero], heap, position, bounded, vertex, themap)
			} else {
				insert(sorted[i].zero, heap, position, bounded, vertex, themap)
				if heap.peek(0) != old_segment {
					shorten = true
				}
			}

			i++
			if i == len(sorted) {
				break
			}

			if sorted[i].two < sorted[orig].two+epsilon {
				break
			}
		}

		if extend {
			polygon = append(polygon, vertex)
			cur, doesintersect := intersectLines(bounded[heap.peek(0)].Points[0], bounded[heap.peek(0)].Points[1], position, vertex)
			if !equal(cur, vertex) && doesintersect {
				polygon = append(polygon, cur)
			}
		} else if shorten {
			if p, doesintersect := intersectLines(bounded[old_segment].Points[0], bounded[old_segment].Points[1], position, vertex); doesintersect {
				polygon = append(polygon, p)
			}
			if p, doesintersect := intersectLines(bounded[heap.peek(0)].Points[0], bounded[heap.peek(0)].Points[1], position, vertex); doesintersect {
				polygon = append(polygon, p)
			}
		}
	}

	return polygon
}

type angledPoint struct {
	zero int
	one  int
	two  float64
}

type byAngle []angledPoint

func (coll byAngle) Len() int      { return len(coll) }
func (coll byAngle) Swap(i, j int) { coll[i], coll[j] = coll[j], coll[i] }
func (coll byAngle) Less(i, j int) bool {
	return coll[i].two < coll[j].two
}

func sortPoints(position [2]float64, segments []Segment) []angledPoint {

	points := make([]angledPoint, len(segments)*2)

	for i := 0; i < len(segments); i++ {
		for j := 0; j < 2; j++ {
			a := angle(segments[i].Points[j], position)
			points[2*i+j] = angledPoint{zero: i, one: j, two: a}
		}
	}

	sort.Sort(byAngle(points))

	return points
}

func angle(a, b [2]float64) float64 {
	return math.Atan2(b[1]-a[1], b[0]-a[0]) * 180 / math.Pi
}

func angle2(a, b, c [2]float64) float64 {
	var a1 = angle(a, b)
	var a2 = angle(b, c)
	var a3 = a1 - a2
	if a3 < 0 {
		a3 += 360
	}
	if a3 > 360 {
		a3 -= 360
	}
	return a3
}

func insert(index int, heap intstack, position [2]float64, segments []Segment, destination [2]float64, themap []int) {
	_, doesintersect := intersectLines(segments[index].Points[0], segments[index].Points[1], position, destination)
	if !doesintersect {
		return
	}

	cur := heap.len()
	heap = append(heap, index)
	themap[index] = cur // TODO: in js, if this is out of boiund it might work anyway; watch that !

	for cur > 0 {
		var parent = getparent(cur)
		if !lessThan(heap.peek(cur), heap.peek(parent), position, segments, destination) {
			break
		}
		themap[heap.peek(parent)] = cur
		themap[heap.peek(cur)] = parent
		var temp = heap.peek(cur)
		heap[cur] = heap[parent]
		heap[parent] = temp
		cur = parent
	}
}

func remove(index int, heap intstack, position [2]float64, segments []Segment, destination [2]float64, themap []int) {
	themap[heap.peek(index)] = -1
	if index == heap.len()-1 {
		heap.pop()
		return
	}

	heap[index] = heap.pop()
	themap[heap.peek(index)] = index
	var cur = index
	var parent = getparent(cur)

	if cur != 0 && lessThan(heap.peek(cur), heap.peek(parent), position, segments, destination) {
		for cur > 0 {
			var parent = getparent(cur)
			if !lessThan(heap.peek(cur), heap.peek(parent), position, segments, destination) {
				break
			}
			themap[heap.peek(parent)] = cur
			themap[heap.peek(cur)] = parent
			var temp = heap.peek(cur)
			heap[cur] = heap.peek(parent)
			heap[parent] = temp
			cur = parent
		}
	} else {
		for true {
			var left = getchild(cur)
			var right = left + 1
			if left < heap.len() && lessThan(heap.peek(left), heap.peek(cur), position, segments, destination) &&
				(right == heap.len() || lessThan(heap.peek(left), heap.peek(right), position, segments, destination)) {
				themap[heap.peek(left)] = cur
				themap[heap.peek(cur)] = left
				var temp = heap.peek(left)
				heap[left] = heap.peek(cur)
				heap[cur] = temp
				cur = left
			} else if right < heap.len() && lessThan(heap.peek(right), heap.peek(cur), position, segments, destination) {
				themap[heap.peek(right)] = cur
				themap[heap.peek(cur)] = right
				var temp = heap.peek(right)
				heap[right] = heap.peek(cur)
				heap[cur] = temp
				cur = right
			} else {
				break
			}
		}
	}
}

func getparent(index int) int {
	res := (index - 1) / 2
	return int(res)
}

func getchild(index int) int {
	return 2*index + 1
}

func lessThan(index1 int, index2 int, position [2]float64, segments []Segment, destination [2]float64) bool {
	inter1, inter1Intersects := intersectLines(segments[index1].Points[0], segments[index1].Points[1], position, destination)
	inter2, inter2Intersects := intersectLines(segments[index2].Points[0], segments[index2].Points[1], position, destination)

	if !inter1Intersects {
		return true
	} else if !inter2Intersects {
		return false
	}

	if !equal(inter1, inter2) {
		var d1 = distance(inter1, position)
		var d2 = distance(inter2, position)
		return d1 < d2
	}

	var end1 = 0
	if equal(inter1, segments[index1].Points[0]) {
		end1 = 1
	}

	var end2 = 0
	if equal(inter2, segments[index2].Points[0]) {
		end2 = 1
	}

	var a1 = angle2(segments[index1].Points[end1], inter1, position)
	var a2 = angle2(segments[index2].Points[end2], inter2, position)
	if a1 < 180 {
		if a2 > 180 {
			return true
		}
		return a2 < a1
	}
	return a1 < a2
}
*/
