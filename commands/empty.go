package commands

func empty() *Maze {
	m := emptyMaze()
	m.addBounds()

	return m
}
