package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/TheGrizzlyDev/git-analyse/bisect"
)

var (
	bisectCmd = flag.NewFlagSet("bisect", flag.ExitOnError)
	jobs      = bisectCmd.Int("jobs", runtime.NumCPU(), "how many jobs can you run concurrently")
	goodRev   = bisectCmd.String("good", "", "last known good revision")
	badRev    = bisectCmd.String("bad", "", "first known bad revision")
)

type command interface {
	Run(ctx context.Context) error
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("expected 'bisect' subcommand")
		os.Exit(1)
	}

	var cmd command
	switch os.Args[1] {
	case "bisect":
		bisectCmd.Parse(os.Args[2:])
		cmd = bisect.NewBisect(bisect.BisectOpts{
			Jobs: *jobs,
			Good: *goodRev,
			Bad:  *badRev,
			Cmd:  bisectCmd.Args(),
		})
	}

	ctx := context.TODO()
	if err := cmd.Run(ctx); err != nil {
		panic(err)
	}
}
