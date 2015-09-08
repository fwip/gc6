package commands

func empty() *Maze {
	m := emptyMaze()
	m.addBounds()
	m.placeRandomly()
	m.SetStartPoint(m.start.X, m.start.Y)
	m.SetTreasure(m.end.X, m.end.Y)

	return m
}
