package bisect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

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
				// https://git-scm.com/docs/git-bisect
				exitCode := l.checkRev(jobCtx, rev.Rev, cmd)
				if exitCode == 0 {
					rev.Good()
				} else if exitCode > 0 && exitCode <= 127 && exitCode != 125 {
					rev.Bad()
				} else if exitCode != -1 {
					// -1 is caused by the context being cancelled and thus should be ignored
					panic(fmt.Sprintf("Unexpected exit code %d", exitCode))
				}
				cancelJob()
			}()
		}
	}()
	go func() {
		ticker := time.NewTicker(time.Second / 24)
		for _ = range ticker.C {
			runState.stats <- bisectState.Stats()
		}
	}()
	return runState
}

func (l *LocalRunner) checkRev(ctx context.Context, rev string, cmd []string) int {
	wpPath := path.Join(settings.BisectWorkspacePath, rev)
	_ = os.Mkdir(wpPath, os.ModePerm)

	trackedFiles, err := gitfsClient.Ls(ctx, rev)
	if err != nil {
		panic(err)
	}
	fsProvisioningJobs := 4
	if fsProvisioningJobs >= len(trackedFiles) {
		fsProvisioningJobs = 1
	}
	fsProvisioningJobsCompleted := make(chan struct{})
	for fsJob := 0; fsJob < fsProvisioningJobs; fsJob++ {
		// TODO clean this up
		go func(fsJob int) {
			for i := fsJob; i < len(trackedFiles); i += fsProvisioningJobs {
				trackedFile := trackedFiles[i]
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
			fsProvisioningJobsCompleted <- struct{}{}
		}(fsJob)
	}
	for fsJob := 0; fsJob < fsProvisioningJobs; fsJob++ {
		<-fsProvisioningJobsCompleted
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

	return runnableCmd.ProcessState.ExitCode()
}
