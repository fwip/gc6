// This keeps track of all the unexplored (accessible) rooms, and at each step just runs toward whichever is closest
package commands

import "github.com/golangchallenge/gc6/mazelib"

type nearest struct {
	memory plot
	pos    mazelib.Coordinate
	path   path
}

func (s *nearest) nextDir(survey mazelib.Survey) int {
	if len(s.path) == 0 {
		s.path = s.memory.ShortestPathToUnexplored(s.pos)
	}
	next := s.path[0]
	s.path = s.path[1:]
	return next
}

func (s *nearest) Solve(surveys <-chan mazelib.Survey, cmds chan<- int) {
	defer close(cmds)

	s.memory = make(map[mazelib.Coordinate]mazelib.Survey)

	for survey := range surveys {
		s.memory[s.pos] = survey

		newdir := s.nextDir(survey)

		s.pos = nextCoord(s.pos, newdir)
		cmds <- newdir
	}
}
