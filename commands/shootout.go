// Shootout benches all solvers against all mazes

package commands

import (
	"fmt"
	"sync"

	"github.com/golangchallenge/gc6/mazelib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type mazeGen func() *Maze
type solverGen func() solver

var gens = []mazeGen{empty, braid, growingTree}
var solvers = []solverGen{newTremaux, newNearest}

var shootoutCmd = &cobra.Command{
	Use:     "shootout",
	Aliases: []string{"bench"},
	Short:   "Bench each solver against each maze type",
	//Long: `Daedalus's job is to create a challenging Labyrinth for his opponent
	//Icarus to solve.
	//
	//Daedalus runs a server which Icarus clients can connect to to solve laybrinths.`,
	Run: func(cmd *cobra.Command, args []string) {
		shootout(gens, solvers)
	},
}

func init() {
	RootCmd.AddCommand(shootoutCmd)
}

func shootout(gens []mazeGen, solvers []solverGen) {
	fmt.Println(gens)
	fmt.Println(solvers)
	times := viper.GetInt("times")

	var w sync.WaitGroup
	results := make([][]int, len(gens))
	for i := range gens {
		results[i] = make([]int, len(solvers))
		for j := range solvers {
			w.Add(1)
			go func(times, i, j int) {
				results[i][j] = fight(gens[i], solvers[j], times)
				w.Done()
			}(times, i, j)
		}
	}
	w.Wait()
	printTable(results)
}

func printTable(table [][]int) {
	if len(table) == 0 {
		return
	}
	for j := range table[0] {
		fmt.Printf("\t%d", j)
	}
	fmt.Print("\n")
	for i := range table {
		fmt.Print(i)
		for j := range table[i] {
			fmt.Printf("\t%d", table[i][j])
		}
		fmt.Print("\n")
	}
}

func fight(gen mazeGen, solver solverGen, times int) (avgSteps int) {
	total := 0
	for i := 0; i < times; i++ {
		m := gen()
		steps := solveIt(m, solver())
		total += steps
	}
	return total / times
}

func solveIt(m *Maze, s solver) int {
	maxSteps := viper.GetInt("max-steps")
	surveys := make(chan mazelib.Survey, 1)
	cmds := make(chan int)
	defer close(surveys)

	go s.Solve(surveys, cmds)
	steps := 0
	for m.icarus != m.end {
		room, _ := m.GetRoom(m.Icarus())
		surveys <- room.Walls
		dir := <-cmds

		steps++
		err := m.moveDir(dir)
		if err != nil {
			fmt.Println(err)
			return 100000000
		}

		if steps > maxSteps {
			return steps
		}
	}

	return steps
}
