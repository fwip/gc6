package commands

import (
	"math/rand"

	"github.com/golangchallenge/gc6/mazelib"
)

func growingTree() *Maze {
	m := fullMaze()

	m.growTree()

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

func (m *Maze) growTree() {
	start := m.randCoord()
	toCarve := []mazelib.Coordinate{start}

	for len(toCarve) > 0 {
		idx := rand.Intn(len(toCarve))
		c := toCarve[idx]

		neighbors := m.unmadeNeighbors(c)
		if len(neighbors) == 0 {
			toCarve = append(toCarve[:idx], toCarve[idx+1:]...)
		} else {
			n := neighbors[rand.Intn(len(neighbors))]
			m.carveTo(c, n)
			toCarve = append(toCarve, n)
		}

	}

}

var offsets = []struct{ X, Y int }{
	{0, -1},
	{0, 1},
	{-1, 0},
	{1, 0},
}

func (m *Maze) unmadeNeighbors(c mazelib.Coordinate) []mazelib.Coordinate {
	var unmade []mazelib.Coordinate
	for _, offset := range offsets {
		neighbor, err := m.GetRoom(c.X+offset.X, c.Y+offset.Y)
		if err != nil {
			continue
		}
		if !isCarved(neighbor.Walls) {
			unmade = append(unmade, mazelib.Coordinate{X: c.X + offset.X, Y: c.Y + offset.Y})
		}
	}
	return unmade
}

func isCarved(svy mazelib.Survey) bool {
	return len(validDirections(svy)) != 0
}

func (m *Maze) carveTo(c1, c2 mazelib.Coordinate) {
	delta := mazelib.Coordinate{
		c1.X - c2.X,
		c1.Y - c2.Y,
	}
	r1, _ := m.getRoomAt(c1)
	r2, _ := m.getRoomAt(c2)
	dir := mazelib.N
	switch delta {
	case mazelib.Coordinate{0, -1}:
		dir = mazelib.S
	case mazelib.Coordinate{0, 1}:
		dir = mazelib.N
	case mazelib.Coordinate{-1, 0}:
		dir = mazelib.E
	case mazelib.Coordinate{1, 0}:
		dir = mazelib.W
	}
	r1.RmWall(dir)
	r2.RmWall(int(direction(dir).Reverse()))
}
