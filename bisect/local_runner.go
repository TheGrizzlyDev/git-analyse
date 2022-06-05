package bisect

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

type LocalRunner struct {
	jobs int
}

func NewLocalRunner(jobs int) *LocalRunner {
	return &LocalRunner{jobs: jobs}
}

func (l *LocalRunner) Run(ctx context.Context, revs []string, cmd []string) *RunnerState {
	runState := NewStartedRunnerState()
	bisectState := NewBisectState(revs)
	jobsGuard := make(chan struct{}, l.jobs)
	runCtx, cancelRun := context.WithCancel(ctx)

	go func() {
		runState.done <- <-bisectState.Done
		cancelRun()
	}()

	go func() {
		for {
			jobsGuard <- struct{}{}
			rev := bisectState.Next()
			if rev == nil {
				break
			}

			jobCtx, cancelJob := context.WithCancel(runCtx)
			go func() {
				<-jobCtx.Done()
				<-jobsGuard
			}()
			go func() {
				<-rev.Cancel
				cancelJob()
			}()
			go func() {
				if l.checkRev(jobCtx, rev.Rev, cmd) {
					rev.Good()
				} else {
					rev.Bad()
				}
				cancelJob()
			}()
		}
	}()
	// go func() {
	// 	ticker := time.NewTicker(time.Second / 24)
	// 	for _ = range ticker.C {
	// 		l.updateState(state)
	// 	}
	// }()
	return runState
}

func (l *LocalRunner) checkRev(ctx context.Context, rev string, cmd []string) bool {
	var out bytes.Buffer

	execCmd := exec.Command("git", "ls-tree", "-r", "--full-name", rev)

	execCmd.Stdout = &out

	if err := execCmd.Run(); err != nil {
		return false
	}

	hashToPath := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		fields := strings.Fields(line)
		hash, path := fields[2], fields[3]
		hashToPath[hash] = path
	}

	return execCmd.ProcessState.ExitCode() == 0
}
