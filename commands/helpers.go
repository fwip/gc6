package commands

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/fwip/gc6/mazelib"
)

type path []int

type plot map[mazelib.Coordinate]mazelib.Survey

func (p *plot) Record(c mazelib.Coordinate, s mazelib.Survey) {
	(*p)[c] = s
}

func (p *plot) ShortestPathToUnexplored(start mazelib.Coordinate) path {
	directions := make(map[mazelib.Coordinate]int)
	//queue := make([]mazelib.Coordinate, 0, 64)
	//queue = append(queue, start)
	queue := []mazelib.Coordinate{start}

	current := start
	for len(queue) > 0 {
		current = queue[0]
		svy, explored := (*p)[current]

		//fmt.Println("Q:", start, current, svy, explored, queue)
		if !explored { // We found where we're going!
			//fmt.Println("Unexp:", current)
			break
		}

		dirs := validDirections(svy)
		for _, dir := range dirs {
			n := nextCoord(current, dir)
			//fmt.Println("dir:", dir, n)
			if _, visited := directions[n]; !visited {
				directions[n] = dir
				queue = append(queue, n)
			}
		}
		queue = queue[1:]
	}

	path := []int{}
	for current != start {
		dir := directions[current]
		path = append([]int{dir}, path...)
		//fmt.Println("Checking", current, dir, path)
		current = nextCoord(current, int(direction(dir).Reverse()))
	}

	return path
}

func isJunction(svy mazelib.Survey) bool {
	return len(validDirections(svy)) > 2
}

func nextCoord(c mazelib.Coordinate, direction int) mazelib.Coordinate {
	out := mazelib.Coordinate{X: c.X, Y: c.Y}
	switch direction {
	case mazelib.N:
		out.Y--
	case mazelib.S:
		out.Y++
	case mazelib.E:
		out.X++
	case mazelib.W:
		out.X--
	}
	return out
}

func validDirections(s mazelib.Survey) []int {
	adjacent := make([]int, 0, 4)
	if !s.Top {
		adjacent = append(adjacent, mazelib.N)
	}
	if !s.Bottom {
		adjacent = append(adjacent, mazelib.S)
	}
	if !s.Left {
		adjacent = append(adjacent, mazelib.W)
	}
	if !s.Right {
		adjacent = append(adjacent, mazelib.E)
	}
	return adjacent
}

func canGo(s mazelib.Survey, dir int) bool {
	switch dir {
	case mazelib.N:
		return !s.Top
	case mazelib.S:
		return !s.Bottom
	case mazelib.E:
		return !s.Right
	case mazelib.W:
		return !s.Left
	}
	return false
}

func (m *Maze) getAdjacent(c mazelib.Coordinate) []mazelib.Coordinate {
	adjacent := make([]mazelib.Coordinate, 0, 4)
	room := m.rooms[c.Y][c.X]
	if !room.Walls.Top {
		adjacent = append(adjacent, mazelib.Coordinate{c.X, c.Y - 1})
	}
	if !room.Walls.Bottom {
		adjacent = append(adjacent, mazelib.Coordinate{c.X, c.Y + 1})
	}
	if !room.Walls.Left {
		adjacent = append(adjacent, mazelib.Coordinate{c.X - 1, c.Y})
	}
	if !room.Walls.Right {
		adjacent = append(adjacent, mazelib.Coordinate{c.X + 1, c.Y})
	}
	return adjacent
}

func (m *Maze) containsOneWayWalls() bool {
	for y := 0; y < m.Height()-1; y++ {
		for x := 0; x < m.Width()-1; x++ {
			if (m.rooms[y][x].Walls.Bottom != m.rooms[y+1][x].Walls.Top) || (m.rooms[y][x].Walls.Right != m.rooms[y][x+1].Walls.Left) {
				return true
			}

		}
	}
	return false
}

func numWalls(r *mazelib.Room) int {
	return 4 - len(validDirections(r.Walls))
}

func (m *Maze) addWall(c mazelib.Coordinate, dir int) (ok bool) {

	c2 := nextCoord(c, dir)
	r1, err1 := m.GetRoom(c.X, c.Y)
	r2, err2 := m.GetRoom(c2.X, c2.Y)

	if err1 != nil || err2 != nil || numWalls(r1) > 1 || numWalls(r2) > 1 {
		fmt.Println(err1, err2, r1, r2, numWalls(r1), numWalls(r2))
		return false
	}
	r1.AddWall(dir)
	r2.AddWall(int(direction(dir).Reverse()))
	return true
}

func (m *Maze) addBounds() {
	xmax := m.Width() - 1
	ymax := m.Height() - 1
	for x := 0; x <= xmax; x++ {
		m.rooms[0][x].AddWall(mazelib.N)
		m.rooms[ymax][x].AddWall(mazelib.S)
	}
	for y := 0; y <= ymax; y++ {
		m.rooms[y][0].AddWall(mazelib.W)
		m.rooms[y][xmax].AddWall(mazelib.E)
	}
}

func (m *Maze) randCoord() mazelib.Coordinate {
	return mazelib.Coordinate{X: rand.Intn(m.Width()), Y: rand.Intn(m.Height())}
}

func (m *Maze) placeRandomly() {
	m.start = m.randCoord()
	m.end = m.randCoord()
}

func (m *Maze) isSolvable() bool {
	return m.isConnected(m.start, m.end, nil)
}

func (m *Maze) isConnected(start, end mazelib.Coordinate, visited map[mazelib.Coordinate]struct{}) bool {
	if start == end {
		return true
	}

	if visited == nil {
		visited = make(map[mazelib.Coordinate]struct{}, 0)
	}

	for _, c := range m.getAdjacent(start) {
		_, seen := visited[c]
		if !seen {
			visited[c] = struct{}{}
			if m.isConnected(c, end, visited) {
				return true
			}
		}
	}
	return false
}

func (m *Maze) getRoomAt(c mazelib.Coordinate) (*mazelib.Room, error) {
	return m.GetRoom(c.X, c.Y)
}

func (m *Maze) moveDir(dir int) error {
	switch dir {
	case mazelib.N:
		return m.MoveUp()
	case mazelib.E:
		return m.MoveRight()
	case mazelib.S:
		return m.MoveDown()
	case mazelib.W:
		return m.MoveLeft()
	}
	return errors.New("Invalid direction")
}
