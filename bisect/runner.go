package bisect

import "context"

type RunnerState struct {
	stats chan *BisectStats
	err   chan error
	done  chan string
}

type BisectStats struct {
	Pending int
	Left    int
	Total   int
}

func NewStartedRunnerState() *RunnerState {
	stats := make(chan *BisectStats)
	err := make(chan error)
	done := make(chan string)

	return &RunnerState{
		stats: stats,
		err:   err,
		done:  done,
	}
}

type Runner interface {
	Run(ctx context.Context, revs []string, cmd []string) *RunnerState
}
