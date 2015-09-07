package commands

import "github.com/golangchallenge/gc6/mazelib"

type tremaux struct {
	memory       map[mazelib.Coordinate]mazelib.Survey
	visited      map[mazelib.Coordinate]int
	pos          mazelib.Coordinate
	dir          int
	backtracking bool
}

func newTremaux() solver {
	return &tremaux{}
}

func (s *tremaux) nextDir(survey mazelib.Survey) int {
	valid := validDirections(survey)
	minDir := valid[0]
	minCost := s.visited[nextCoord(s.pos, minDir)]

	for _, dir := range valid[1:] {
		cost := s.visited[nextCoord(s.pos, dir)]
		if cost < minCost {
			minDir = dir
			minCost = cost
		}
	}

	return minDir
}

func (s *tremaux) Solve(surveys <-chan mazelib.Survey, cmds chan<- int) {
	defer close(cmds)
	s.dir = mazelib.N

	s.memory = make(map[mazelib.Coordinate]mazelib.Survey)
	s.visited = make(map[mazelib.Coordinate]int)

	for survey := range surveys {
		s.visited[s.pos]++
		s.memory[s.pos] = survey

		newdir := s.nextDir(survey)
		ahead := nextCoord(s.pos, newdir)
		if !s.backtracking && s.visited[ahead] > 0 {
			newdir = int(direction(s.dir).Reverse())
			s.backtracking = true
		}

		if int(direction(newdir).Reverse()) == s.dir {
			s.visited[s.pos]++
		}

		if s.visited[ahead] == 0 {
			s.backtracking = false
		}

		s.dir = newdir
		s.pos = nextCoord(s.pos, s.dir)
		cmds <- s.dir
	}
}
