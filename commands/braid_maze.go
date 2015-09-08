package commands

import (
	"math/rand"

	"github.com/golangchallenge/gc6/mazelib"
)

func braid() *Maze {
	m := emptyMaze()
	m.addBounds()

	m.braidFill()

	m.placeRandomly()
	for !m.isSolvable() {
		m.placeRandomly()
	}
	m.SetStartPoint(m.start.X, m.start.Y)
	m.SetTreasure(m.end.X, m.end.Y)

	if m.containsOneWayWalls() {
		panic("Oh no! One way walls!")
	}

	return m
}

func (m *Maze) braidFill() {
	for wallCount := 0; wallCount < 550; wallCount++ {
		loc := m.randCoord()
		dir := mazelib.E
		if rand.Intn(2) == 1 {
			dir = mazelib.S
		}
		loc2 := nextCoord(loc, dir)
		r1, err1 := m.GetRoom(loc.X, loc.Y)
		r2, err2 := m.GetRoom(loc2.X, loc2.Y)

		if err1 != nil || err2 != nil || numWalls(r1) > 1 || numWalls(r2) > 1 {
			continue
		} else {
			r1.AddWall(dir)
			r2.AddWall(int(direction(dir).Reverse()))
		}
	}
}