package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/TheGrizzlyDev/git-analyse/bisect"
	_ "github.com/TheGrizzlyDev/git-analyse/settings"
)

var (
	bisectCmd = flag.NewFlagSet("bisect", flag.ExitOnError)
	jobs      = bisectCmd.Int("jobs", runtime.NumCPU(), "how many jobs can you run concurrently")
	goodRev   = bisectCmd.String("good", "", "last known good revision")
	badRev    = bisectCmd.String("bad", "", "first known bad revision")

	// internal
	profiled = ""
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

	if profiled != "" {
		fmt.Printf("Profiling on %s\n", profiled)
		f, err := os.Create(profiled)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)

		defer pprof.StopCPUProfile()
	}

	ctx := context.TODO()
	if err := cmd.Run(ctx); err != nil {
		panic(err)
	}
}
