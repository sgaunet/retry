// Package retry provides a simple way to retry a command execution based on a condition.
package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"github.com/go-andiamo/splitter"
)

var (
	// ErrConditionNil is returned when the condition is nil.
	ErrConditionNil = errors.New("condition is nil")
	// ErrMaxTriesReached is returned when the maximum number of tries is reached.
	ErrMaxTriesReached = errors.New("max tries reached")
)

// Retry is a struct that represents a retry mechanism for executing commands.
type Retry struct {
	cmd       string
	sleep     func()
	tries     int
	condition ConditionRetryer
}

// ConditionRetryer is an interface that defines the methods required for a retry condition.
type ConditionRetryer interface {
	GetCtx() context.Context
	IsLimitReached() bool
	StartTry()
	EndTry()
}

// NewRetry creates a new retry instance with the given command and condition.
func NewRetry(cmd string, condition ConditionRetryer) (*Retry, error) {
	r := &Retry{
		cmd:       cmd,
		sleep:     nil,
		condition: condition,
	}
	if r.condition == nil {
		return nil, ErrConditionNil
	}
	return r, nil
}

// SetSleep sets the sleep function to be used between retries.
func (r *Retry) SetSleep(sleep func()) {
	r.sleep = sleep
}

// Run executes the command with retries based on the condition.
// It returns an error if the command fails or if the maximum number of tries is reached.
// It also logs the output of the command to the provided logger.
func (r *Retry) Run(logger *slog.Logger) error {
	var err error

	for r.shouldContinue() {
		err = r.executeSingleTry(logger)
		if err == nil {
			logger.Info("Command executed successfully")
			break
		}
		if r.sleep != nil {
			r.sleep()
		}
	}

	return r.getFinalError(err)
}

// shouldContinue checks if the retry loop should continue.
func (r *Retry) shouldContinue() bool {
	return r.condition.GetCtx().Err() == nil && !r.condition.IsLimitReached()
}

// executeSingleTry executes a single retry attempt.
func (r *Retry) executeSingleTry(logger *slog.Logger) error {
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
	
	return err
}

// getFinalError determines the final error to return.
func (r *Retry) getFinalError(err error) error {
	if r.condition.GetCtx().Err() != nil {
		return fmt.Errorf("context error: %w", r.condition.GetCtx().Err())
	}
	if r.condition.IsLimitReached() && err != nil {
		return ErrMaxTriesReached
	}
	return err
}

func execCommand(ctx context.Context, cmd string) (int, error) {
	var wg sync.WaitGroup
	nbGoroutines := 2
	commandSplitter, _ := splitter.NewSplitter(' ', splitter.SingleQuotes, splitter.DoubleQuotes)
	trimmer := splitter.Trim("'\"")
	splitCmd, _ := commandSplitter.Split(cmd, trimmer)
	if len(splitCmd) == 0 {
		return -1, nil
	}
	c := exec.CommandContext(ctx, splitCmd[0], splitCmd[1:]...) //nolint:gosec

	stderr, err := c.StderrPipe()
	if err != nil {
		return -1, fmt.Errorf("error creating stderr pipe: %w", err)
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return -1, fmt.Errorf("error creating stdout pipe: %w", err)
	}

	err = c.Start()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			// The command didn't exit successfully, so we can get the exit code
			return exitError.ExitCode(), fmt.Errorf("command failed: %w", err)
		}
		// The command didn't start at all or exited because of a signal
		return -1, fmt.Errorf("command failed: %w", err)
	}

	wg.Add(nbGoroutines)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stdout, stdout)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stderr, stderr)
	}()

	err = c.Wait()
	stderr.Close() //nolint:errcheck,gosec
	stdout.Close() //nolint:errcheck,gosec
	wg.Wait()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			// The command didn't exit successfully, so we can get the exit code
			return exitError.ExitCode(), fmt.Errorf("command failed: %w", err)
		}
		// The command didn't start at all or exited because of a signal
		return -1, fmt.Errorf("command failed: %w", err)
	}
	return 0, nil
}
