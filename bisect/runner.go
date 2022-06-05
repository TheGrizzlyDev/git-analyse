package bisect

import "context"

type RunnerState struct {
	left chan []string
	wip  chan []string
	err  chan error
	done chan string
}

func NewStartedRunnerState() *RunnerState {
	left := make(chan []string)
	wip := make(chan []string)
	err := make(chan error)
	done := make(chan string)

	return &RunnerState{
		left: left,
		wip:  wip,
		err:  err,
		done: done,
	}
}

type Runner interface {
	Run(ctx context.Context, revs []string, cmd []string) *RunnerState
}
