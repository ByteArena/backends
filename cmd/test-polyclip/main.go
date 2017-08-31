package main

import (
	"log"

	polyclip "github.com/akavel/polyclip-go"
)

func main() {
	subject := polyclip.Polygon{{{2, 3}, {14, 7}, {13, 12}, {0, 6}}}  // small square
	clipping := polyclip.Polygon{{{4, 0}, {8, 0}, {12, 16}, {9, 16}}} // overlapping triangle
	result := subject.Construct(polyclip.INTERSECTION, clipping)

	log.Println(result)
}
