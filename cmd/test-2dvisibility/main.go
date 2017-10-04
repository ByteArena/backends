package main

import "github.com/davecgh/go-spew/spew"

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

	lightSource := MakePoint(100, 100)

	endpoints := loadMap(room, blocks, walls, lightSource)
	spew.Dump(endpoints)
}
