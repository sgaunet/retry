package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sgaunet/retry/pkg/retry"
)

var version = "dev"

func printVersion() {
	fmt.Printf("%s\n", version)
}

func main() {
	var (
		maxtries           uint
		sleepTimeInSeconds uint
		command            string
		printVersionFlag   bool
		helpFlag           bool
	)
	flag.UintVar(&maxtries, "m", 3, "max tries of execution of failed command (0 for infinite tries)")
	flag.UintVar(&sleepTimeInSeconds, "s", 0, "sleep time in seconds between each try")
	flag.StringVar(&command, "c", "", "command to execute")
	flag.BoolVar(&printVersionFlag, "version", false, "print version")
	flag.BoolVar(&helpFlag, "h", false, "print help")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	if printVersionFlag {
		printVersion()
		os.Exit(0)
	}

	if command == "" {
		fmt.Fprintln(os.Stderr, "command is empty")
		flag.Usage()
		os.Exit(1)
	}

	r, err := retry.NewRetry(command, retry.NewStopOnMaxTries(maxtries))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	r.SetSleep(func() { time.Sleep(time.Duration(sleepTimeInSeconds) * time.Second) })
	err = r.Run(logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
