// Copyright © 2015 Steve Francia <spf@spf13.com>.
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//

package commands

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golangchallenge/gc6/mazelib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Maze struct {
	rooms      [][]mazelib.Room
	start      mazelib.Coordinate
	end        mazelib.Coordinate
	icarus     mazelib.Coordinate
	StepsTaken int
}

type direction byte

func (d direction) Reverse() direction {
	switch d {
	case mazelib.N:
		return mazelib.S
	case mazelib.S:
		return mazelib.N
	case mazelib.E:
		return mazelib.W
	case mazelib.W:
		return mazelib.E
	}
	panic("Not a direction")
}

// Tracking the current maze being solved

// WARNING: This approach is not safe for concurrent use
// This server is only intended to have a single client at a time
// We would need a different and more complex approach if we wanted
// concurrent connections than these simple package variables
var currentMaze *Maze
var scores []int

// Defining the daedalus command.
// This will be called as 'laybrinth daedalus'
var daedalusCmd = &cobra.Command{
	Use:     "daedalus",
	Aliases: []string{"deadalus", "server"},
	Short:   "Start the laybrinth creator",
	Long: `Daedalus's job is to create a challenging Labyrinth for his opponent
  Icarus to solve.

  Daedalus runs a server which Icarus clients can connect to to solve laybrinths.`,
	Run: func(cmd *cobra.Command, args []string) {
		RunServer()
	},
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) // need to initialize the seed
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = noop{}

	RootCmd.AddCommand(daedalusCmd)
}

// Runs the web server
func RunServer() {
	// Adding handling so that even when ctrl+c is pressed we still print
	// out the results prior to exiting.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		printResults()
		os.Exit(1)
	}()

	// Using gin-gonic/gin to handle our routing
	r := gin.Default()
	v1 := r.Group("/")
	{
		v1.GET("/awake", GetStartingPoint)
		v1.GET("/move/:direction", MoveDirection)
		v1.GET("/done", End)
	}

	r.Run(":" + viper.GetString("port"))
}

// Ends a session and prints the results.
// Called by Icarus when he has reached
//   the number of times he wants to solve the laybrinth.
func End(c *gin.Context) {
	printResults()
	os.Exit(1)
}

// initializes a new maze and places Icarus in his awakening location
func GetStartingPoint(c *gin.Context) {
	initializeMaze()
	startRoom, err := currentMaze.Discover(currentMaze.Icarus())
	if err != nil {
		fmt.Println("Icarus is outside of the maze. This shouldn't ever happen")
		fmt.Println(err)
		os.Exit(-1)
	}
	mazelib.PrintMaze(currentMaze)

	c.JSON(http.StatusOK, mazelib.Reply{Survey: startRoom})
}

// The API response to the /move/:direction address
func MoveDirection(c *gin.Context) {
	var err error

	switch c.Param("direction") {
	case "left":
		err = currentMaze.MoveLeft()
	case "right":
		err = currentMaze.MoveRight()
	case "down":
		err = currentMaze.MoveDown()
	case "up":
		err = currentMaze.MoveUp()
	}

	var r mazelib.Reply

	if err != nil {
		r.Error = true
		r.Message = err.Error()
		c.JSON(409, r)
		return
	}

	s, e := currentMaze.LookAround()

	if e != nil {
		if e == mazelib.ErrVictory {
			scores = append(scores, currentMaze.StepsTaken)
			r.Victory = true
			r.Message = fmt.Sprintf("Victory achieved in %d steps \n", currentMaze.StepsTaken)
		} else {
			r.Error = true
			r.Message = err.Error()
		}
	}

	r.Survey = s

	c.JSON(http.StatusOK, r)
}

func initializeMaze() {
	currentMaze = createMaze()
}

// Print to the terminal the average steps to solution for the current session
func printResults() {
	fmt.Printf("Labyrinth solved %d times with an avg of %d steps\n", len(scores), mazelib.AvgScores(scores))
}

// Return a room from the maze
func (m *Maze) GetRoom(x, y int) (*mazelib.Room, error) {
	if x < 0 || y < 0 || x >= m.Width() || y >= m.Height() {
		return &mazelib.Room{}, errors.New("room outside of maze boundaries")
	}

	return &m.rooms[y][x], nil
}

func (m *Maze) Width() int  { return len(m.rooms[0]) }
func (m *Maze) Height() int { return len(m.rooms) }

// Return Icarus's current position
func (m *Maze) Icarus() (x, y int) {
	return m.icarus.X, m.icarus.Y
}

// Set the location where Icarus will awake
func (m *Maze) SetStartPoint(x, y int) error {
	r, err := m.GetRoom(x, y)

	if err != nil {
		return err
	}

	if r.Treasure {
		return errors.New("can't start in the treasure")
	}

	r.Start = true
	m.icarus = mazelib.Coordinate{x, y}
	return nil
}

// Set the location of the treasure for a given maze
func (m *Maze) SetTreasure(x, y int) error {
	r, err := m.GetRoom(x, y)

	if err != nil {
		return err
	}

	if r.Start {
		return errors.New("can't have the treasure at the start")
	}

	r.Treasure = true
	m.end = mazelib.Coordinate{x, y}
	return nil
}

// Given Icarus's current location, Discover that room
// Will return ErrVictory if Icarus is at the treasure.
func (m *Maze) LookAround() (mazelib.Survey, error) {
	if m.end.X == m.icarus.X && m.end.Y == m.icarus.Y {
		return mazelib.Survey{}, mazelib.ErrVictory
	}

	return m.Discover(m.icarus.X, m.icarus.Y)
}

// Given two points, survey the room.
// Will return error if two points are outside of the maze
func (m *Maze) Discover(x, y int) (mazelib.Survey, error) {
	if r, err := m.GetRoom(x, y); err != nil {
		return mazelib.Survey{}, nil
	} else {
		return r.Walls, nil
	}
}

// Moves Icarus's position left one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveLeft() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Left {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x-1, y); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x - 1, y}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position right one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveRight() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Right {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x+1, y); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x + 1, y}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position up one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveUp() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Top {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x, y-1); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x, y - 1}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position down one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveDown() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Bottom {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x, y+1); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x, y + 1}
	m.StepsTaken++
	return nil
}

// Creates a maze without any walls
// Good starting point for additive algorithms
func emptyMaze() *Maze {
	z := Maze{}
	ySize := viper.GetInt("height")
	xSize := viper.GetInt("width")

	z.rooms = make([][]mazelib.Room, ySize)
	for y := 0; y < ySize; y++ {
		z.rooms[y] = make([]mazelib.Room, xSize)
		for x := 0; x < xSize; x++ {
			z.rooms[y][x] = mazelib.Room{}
		}
	}

	return &z
}

// Creates a maze with all walls
// Good starting point for subtractive algorithms
func fullMaze() *Maze {
	z := emptyMaze()
	ySize := viper.GetInt("height")
	xSize := viper.GetInt("width")

	for y := 0; y < ySize; y++ {
		for x := 0; x < xSize; x++ {
			z.rooms[y][x].Walls = mazelib.Survey{true, true, true, true}
		}
	}

	return z
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

// TODO: Write your maze creator function here
func createMaze() *Maze {

	// TODO: Fill in the maze:
	// You need to insert a startingPoint for Icarus
	// You need to insert an EndingPoint (treasure) for Icarus
	// You need to Add and Remove walls as needed.
	// Use the mazelib.AddWall & mazelib.RmWall to do this

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
