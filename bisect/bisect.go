package bisect

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type BisectOpts struct {
	Jobs int
	Good string
	Bad  string
	Cmd  []string
}

type bisect struct {
	jobs   int
	good   string
	bad    string
	cmd    []string
	runner Runner
}

func NewBisect(opts BisectOpts) *bisect {
	runner := NewLocalRunner()
	return &bisect{
		jobs:   opts.Jobs,
		good:   opts.Good,
		bad:    opts.Bad,
		runner: runner,
	}
}

func (b bisect) Run(ctx context.Context) error {
	var gitLogOut bytes.Buffer
	gitLog := exec.CommandContext(ctx, "git", "log", "--format=%h", "--ancestry-path", fmt.Sprintf("%s~1..%s", b.good, b.bad))
	gitLog.Stdout = &gitLogOut
	if err := gitLog.Run(); err != nil {
		return fmt.Errorf("could not get the list of revisions: %v", err)
	}

	revisions := strings.Split(gitLogOut.String(), "\n")
	runState := b.runner.Run(ctx, revisions, b.cmd)

	for {
		select {
		case wip := <-runState.wip:
			fmt.Printf("WIP (%d): %s\n", len(wip), wip)
		case left := <-runState.left:
			fmt.Printf("Left (%d): %s\n", len(left), left)
		case err := <-runState.err:
			return err
		case found := <-runState.done:
			fmt.Println("Found revision: ", found)
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}
