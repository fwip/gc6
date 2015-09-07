// Sometimes you just need something that does nothing

package commands

import "github.com/golangchallenge/gc6/mazelib"

type noop struct{}

func (s noop) Solve(surveys <-chan mazelib.Survey, cmds chan<- int) {
	close(cmds)
}

func (s noop) Write(p []byte) (n int, err error) { return len(p), nil }
