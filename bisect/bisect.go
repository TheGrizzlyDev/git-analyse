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
	runner := NewLocalRunner(opts.Jobs)
	return &bisect{
		jobs:   opts.Jobs,
		good:   opts.Good,
		bad:    opts.Bad,
		cmd:    opts.Cmd,
		runner: runner,
	}
}

func (b bisect) Run(ctx context.Context) error {
	var gitLogOut bytes.Buffer
	gitLog := exec.CommandContext(ctx, "git", "log", "--format=%H", "--ancestry-path", fmt.Sprintf("%s..%s", b.good, b.bad))
	gitLog.Stdout = &gitLogOut
	if err := gitLog.Run(); err != nil {
		return fmt.Errorf("could not get the list of revisions: %v", err)
	}

	revisions := append(strings.Split(strings.TrimSpace(gitLogOut.String()), "\n"), b.good)
	runState := b.runner.Run(ctx, revisions, b.cmd)

	fmt.Println()
	for {
		select {
		case stats := <-runState.stats:
			fmt.Printf("\033[1A\033[K%d revisions left out of %d\n", stats.Left, stats.Total)
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
