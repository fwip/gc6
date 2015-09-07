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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golangchallenge/gc6/mazelib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var dirName = map[int]string{
	mazelib.N: "up",
	mazelib.S: "down",
	mazelib.W: "left",
	mazelib.E: "right",
}

type solver interface {
	Solve(<-chan mazelib.Survey, chan<- int)
}

// Defining the icarus command.
// This will be called as 'laybrinth icarus'
var icarusCmd = &cobra.Command{
	Use:     "icarus",
	Aliases: []string{"client"},
	Short:   "Start the laybrinth solver",
	Long: `Icarus wakes up to find himself in the middle of a Labyrinth.
  Due to the darkness of the Labyrinth he can only see his immediate cell and if
  there is a wall or not to the top, right, bottom and left. He takes one step
  and then can discover if his new cell has walls on each of the four sides.

  Icarus can connect to a Daedalus and solve many laybrinths at a time.`,
	Run: func(cmd *cobra.Command, args []string) {
		RunIcarus()
	},
}

func init() {
	RootCmd.AddCommand(icarusCmd)
}

func RunIcarus() {
	// Run the solver as many times as the user desires.
	fmt.Println("Solving", viper.GetInt("times"), "times")
	for x := 0; x < viper.GetInt("times"); x++ {

		solveMaze()
	}

	// Once we have solved the maze the required times, tell daedalus we are done
	makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/done")
}

// Make a call to the laybrinth server (daedalus) that icarus is ready to wake up
func awake() mazelib.Survey {
	contents, err := makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/awake")
	if err != nil {
		fmt.Println(err)
	}
	r := ToReply(contents)
	return r.Survey
}

// Make a call to the laybrinth server (daedalus)
// to move Icarus a given direction
// Will be used heavily by solveMaze
func Move(direction string) (mazelib.Survey, error) {
	if direction == "left" || direction == "right" || direction == "up" || direction == "down" {

		contents, err := makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/move/" + direction)
		if err != nil {
			return mazelib.Survey{}, err
		}

		rep := ToReply(contents)
		if rep.Victory == true {
			fmt.Println(rep.Message)
			// os.Exit(1)
			return rep.Survey, mazelib.ErrVictory
		} else {
			return rep.Survey, errors.New(rep.Message)
		}
	}

	return mazelib.Survey{}, errors.New("invalid direction")
}

// utility function to wrap making requests to the daedalus server
func makeRequest(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

// Handling a JSON response and unmarshalling it into a reply struct
func ToReply(in []byte) mazelib.Reply {
	res := &mazelib.Reply{}
	json.Unmarshal(in, &res)
	return *res
}

type tremaux struct {
	memory       map[mazelib.Coordinate]mazelib.Survey
	visited      map[mazelib.Coordinate]int
	pos          mazelib.Coordinate
	dir          int
	backtracking bool
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

func isJunction(svy mazelib.Survey) bool {
	return len(validDirections(svy)) > 2
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

func runSolver(s solver) {
	surveys := make(chan mazelib.Survey, 1)
	cmds := make(chan int)
	defer close(surveys)

	var err error
	survey := awake()
	surveys <- survey
	go s.Solve(surveys, cmds)

	maxSteps := viper.GetInt("max-steps")
	steps := 0

	for dir := range cmds {
		name, ok := dirName[dir]
		if !ok {
			fmt.Println("Solver returned", dir, ", not N S E W (1-4)")
			return
		}

		survey, err = Move(name)

		if err.Error() != "" {
			fmt.Println("Error!", err)
			return
		}
		surveys <- survey
		steps++
		if steps > maxSteps {
			fmt.Printf("Reached max-steps (%d), halting\n", maxSteps)
			return
		}

	}

}

type noop struct{}

func (s noop) Solve(surveys <-chan mazelib.Survey, cmds chan<- int) {
	close(cmds)
}

func (s noop) Write(p []byte) (n int, err error) { return len(p), nil }

// TODO: This is where you work your magic
func solveMaze() {
	//_ = awake() // Need to start with waking up to initialize a new maze
	// You'll probably want to set this to a named value and start by figuring
	// out which step to take next
	runSolver(&tremaux{})

}
