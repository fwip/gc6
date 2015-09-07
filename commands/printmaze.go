package commands

import (
	"github.com/golangchallenge/gc6/mazelib"
	"github.com/spf13/cobra"
)

var printMazeCmd = &cobra.Command{
	Use:     "printmaze",
	Aliases: []string{"generate"},
	Short:   "Generate and print a maze",
	Long:    `Daedalus generates a maze, prints it, then exits`,
	Run: func(cmd *cobra.Command, args []string) {
		mazelib.PrintMaze(createMaze())
	},
}

func init() {
	RootCmd.AddCommand(printMazeCmd)
}
