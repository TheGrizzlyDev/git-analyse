package bisect

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/TheGrizzlyDev/git-analyse/gitfs"
	"github.com/TheGrizzlyDev/git-analyse/settings"
)

var (
	gitfsClient = gitfs.New()
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
	wpPath := path.Join(settings.BisectWorkspacePath, rev)
	_ = os.Mkdir(wpPath, os.ModePerm)
	defer os.RemoveAll(wpPath)

	trackedFiles, err := gitfsClient.Ls(ctx, rev)
	if err != nil {
		panic(err)
	}
	for _, trackedFile := range trackedFiles {
		// TODO parallelize this
		// TODO clean this up
		dest := path.Join(wpPath, trackedFile.Path)

		if _, err := os.Stat(dest); err == nil {
			continue
		} else if trackedFile.Mode.IsDir() {
			os.MkdirAll(dest, os.ModePerm)
		} else if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
			parent := filepath.Dir(dest)
			if _, err := os.Stat(parent); err != nil && errors.Is(err, os.ErrNotExist) {
				os.MkdirAll(parent, os.ModePerm)
			}
			err := trackedFile.Link(ctx, dest)
			if err != nil {
				panic(err)
			}
		}
	}

	var out bytes.Buffer

	// TODO If the context expires this will send a sigkill which would change the exit code
	runnableCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	runnableCmd.Dir = wpPath
	runnableCmd.Stdout = &out
	runnableCmd.Run()

	// TODO push logs
	// if out.Len() > 0 {
	// 	doSomething()
	// }

	return runnableCmd.ProcessState.ExitCode() == 0
}
