package bisect

import (
	"context"
	"sync"
	"time"
)

type LocalRunner struct {
	mu      sync.RWMutex
	left    []string
	wip     []string
	lastBad string
	cmd     []string
}

func NewLocalRunner() *LocalRunner {
	return &LocalRunner{}
}

func (l *LocalRunner) Run(ctx context.Context, revs []string, cmd []string) *RunnerState {
	state := NewStartedRunnerState(revs)
	l.cmd = cmd
	l.left = revs
	l.lastBad = revs[len(revs)-1]
	l.wip = []string{}
	go func() {
		l.start(ctx, state)
	}()
	go func() {
		ticker := time.NewTicker(time.Second / 24)
		for _ = range ticker.C {
			l.updateState(state)
		}
	}()
	return state
}

func (l *LocalRunner) updateState(state *RunnerState) {
	if len(l.left) == 0 && len(l.wip) == 0 {
		state.done <- l.lastBad
		return
	}
	state.left <- l.left
	state.wip <- l.wip
}

func (l *LocalRunner) bad(ctx context.Context, rev string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// TODO figure out how to mark a bad hit
	l.lastBad = rev
}

func (l *LocalRunner) good(ctx context.Context, rev string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// TODO figure out how to mark a good hit
}

func (l *LocalRunner) markWip(ctx context.Context, rev string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.wip = append(l.wip, rev)
}

func (l *LocalRunner) start(ctx context.Context, state *RunnerState) {
	// exec.CommandContext(ctx, "TODO", "TODO")

	for i := len(l.left) - 1; i >= 0; i-- {
		l.mu.RLock()
		rev := l.left[i]
		l.mu.RUnlock()
		l.markWip(ctx, rev)
		if i > 5 {
			l.bad(ctx, rev)
		} else {
			l.good(ctx, rev)
		}
		_ = <-time.NewTimer(time.Second).C
	}
}
