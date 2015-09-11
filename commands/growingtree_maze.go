package commands

import (
	"math/rand"

	"github.com/fwip/gc6/mazelib"
)

func growingTree() *Maze {
	m := fullMaze()
	m.growTree(100)

	return m
}

func growingTree20() *Maze {
	m := fullMaze()
	m.growTree(20)
	return m
}

func (m *Maze) growTree(prob int) {
	start := m.randCoord()
	toCarve := []mazelib.Coordinate{start}

	for len(toCarve) > 0 {

		idx := 0
		if rand.Intn(100) < prob {
			idx = rand.Intn(len(toCarve))
		}
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
