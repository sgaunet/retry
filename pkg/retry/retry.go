package retry

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"github.com/go-andiamo/splitter"
)

type retry struct {
	cmd       string
	sleep     func()
	tries     int
	condition ConditionRetryer
}

type ConditionRetryer interface {
	GetCtx() context.Context
	IsLimitReached() bool
	StartTry()
	EndTry()
}

func NewRetry(cmd string, condition ConditionRetryer) (*retry, error) {
	r := &retry{
		cmd:       cmd,
		sleep:     nil,
		condition: condition,
	}
	if r.condition == nil {
		return nil, fmt.Errorf("condition is nil")
	}
	return r, nil
}

func (r *retry) SetSleep(sleep func()) {
	r.sleep = sleep
}

func (r *retry) Run(logger *slog.Logger) error {
	var (
		err error
	)
	for r.condition.GetCtx().Err() == nil && !r.condition.IsLimitReached() {
		if r.condition != nil {
			r.condition.StartTry()
		}
		r.tries++
		logger.Info("Try:", slog.Int("attempt n°", r.tries))
		rc, err := execCommand(r.condition.GetCtx(), r.cmd)
		if rc == 0 {
			logger.Info("End", slog.Int("return code", rc))
		} else {
			logger.Error("End", slog.Int("return code", rc))
		}

		if r.condition != nil {
			r.condition.EndTry()
		}
		if err == nil {
			logger.Info("Command executed successfully")
			break
		}
		if r.sleep != nil {
			r.sleep()
		}
	}
	if r.condition.GetCtx().Err() != nil {
		err = r.condition.GetCtx().Err()
	}
	if r.condition.IsLimitReached() {
		err = fmt.Errorf("max tries reached")
	}
	return err
}

func execCommand(ctx context.Context, cmd string) (int, error) {
	commandSplitter, _ := splitter.NewSplitter(' ', splitter.SingleQuotes, splitter.DoubleQuotes)
	trimmer := splitter.Trim("'\"")
	splitCmd, _ := commandSplitter.Split(cmd, trimmer)
	if len(splitCmd) == 0 {
		return -1, nil
	}
	c := exec.CommandContext(ctx, splitCmd[0], splitCmd[1:]...) //nolint:gosec

	stderr, err := c.StderrPipe()
	if err != nil {
		return -1, err
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return -1, err
	}

	// print output in real time (both stdout and stderr)
	go printOutput(stdout, os.Stdout)
	go printOutput(stderr, os.Stderr)

	err = c.Start()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// The command didn't exit successfully, so we can get the exit code
			return exitError.ExitCode(), err
		}
		// The command didn't start at all or exited because of a signal
		return -1, err
	}
	err = c.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// The command didn't exit successfully, so we can get the exit code
			return exitError.ExitCode(), err
		}
		// The command didn't start at all or exited because of a signal
		return -1, err
	}
	return 0, nil
}

func printOutput(r io.Reader, w io.Writer) {
	var reader = bufio.NewScanner(r)
	for reader.Scan() {
		_, _ = fmt.Fprintln(w, reader.Text())
	}
}
