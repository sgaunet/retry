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
	"strings"
	"sync"
	"time"

	"github.com/go-andiamo/splitter"
)

var (
	// ErrConditionNil is returned when the condition is nil.
	ErrConditionNil = errors.New("condition is nil")
	// ErrMaxTriesReached is returned when the maximum number of tries is reached.
	ErrMaxTriesReached = errors.New("max tries reached")
	// ErrEmptyCommand is returned when the command is empty.
	ErrEmptyCommand = errors.New("empty command")
)

const (
	// outputStreams represents the number of output streams (stdout and stderr).
	outputStreams = 2
)

// Retry is a struct that represents a retry mechanism for executing commands.
type Retry struct {
	cmd          string
	tries        int
	condition    ConditionRetryer
	backoff      BackoffStrategy
	lastExitCode int
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
		condition: condition,
	}
	if r.condition == nil {
		return nil, ErrConditionNil
	}
	return r, nil
}

// SetBackoffStrategy sets the backoff strategy to be used between retries.
func (r *Retry) SetBackoffStrategy(backoff BackoffStrategy) {
	r.backoff = backoff
}

// Run executes the command with retries based on the condition.
// It returns an error if the command fails or if the maximum number of tries is reached.
// It also logs the output of the command to the provided logger.
func (r *Retry) Run(_ *slog.Logger) error {
	return r.RunWithEnhancedLogger(nil)
}


// RunWithEnhancedLogger executes the command with enhanced logging support.
func (r *Retry) RunWithEnhancedLogger(logger *Logger) error {
	r.initializeLogging(logger)
	
	err := r.executeRetryLoop(logger)
	
	failureReason, stopCondition := r.determineFailureReason()
	
	if logger != nil {
		logger.EndExecution(err == nil, failureReason, stopCondition)
	}

	return r.getFinalError(err)
}

// shouldContinue checks if the retry loop should continue.
func (r *Retry) shouldContinue() bool {
	return r.condition.GetCtx().Err() == nil && !r.condition.IsLimitReached()
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


// performBackoffWithDelay handles the delay and returns the delay duration.
func (r *Retry) performBackoffWithDelay() time.Duration {
	if r.backoff != nil {
		delay := r.backoff.NextDelay(r.tries)
		if delay > 0 {
			time.Sleep(delay)
			return delay
		}
	}
	return 0
}

// getLastExitCode returns the exit code from the last command execution.
func (r *Retry) getLastExitCode() int {
	return r.lastExitCode
}

// executeSingleTryWithLogger executes a single retry attempt with enhanced logging.
func (r *Retry) executeSingleTryWithLogger(logger *Logger) error {
	if r.condition != nil {
		r.condition.StartTry()
	}
	r.tries++
	
	rc, stdout, stderr, err := execCommandWithOutputAndLogger(r.condition.GetCtx(), r.cmd, logger)
	r.lastExitCode = rc

	// Pass exit code and output to enhanced conditions
	if enhanced, ok := r.condition.(EnhancedConditionRetryer); ok {
		enhanced.SetLastExitCode(rc)
		enhanced.SetLastOutput(stdout, stderr)
	}

	if r.condition != nil {
		r.condition.EndTry()
	}
	
	return err
}



// parseCommand splits the command string into executable parts.
func parseCommand(cmd string) ([]string, error) {
	commandSplitter, _ := splitter.NewSplitter(' ', splitter.SingleQuotes, splitter.DoubleQuotes)
	trimmer := splitter.Trim("'\"")
	splitCmd, _ := commandSplitter.Split(cmd, trimmer)
	if len(splitCmd) == 0 {
		return nil, ErrEmptyCommand
	}
	return splitCmd, nil
}

// setupCommandPipes creates and returns stdout and stderr pipes for the command.
func setupCommandPipes(c *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating stderr pipe: %w", err)
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		_ = stderr.Close()
		return nil, nil, fmt.Errorf("error creating stdout pipe: %w", err)
	}

	return stdout, stderr, nil
}

// handleCommandOutput processes command output with optional logging.
func handleCommandOutput(stdout, stderr io.ReadCloser, logger *Logger, wg *sync.WaitGroup) (string, string) {
	var stdoutBuf, stderrBuf strings.Builder
	
	wg.Add(outputStreams)
	
	go func() {
		defer wg.Done()
		if logger != nil {
			stdoutWriter := NewPrefixWriter(logger, false)
			_, _ = io.Copy(io.MultiWriter(&stdoutBuf, stdoutWriter), stdout)
			stdoutWriter.Flush()
		} else {
			_, _ = io.Copy(io.MultiWriter(os.Stdout, &stdoutBuf), stdout)
		}
	}()

	go func() {
		defer wg.Done()
		if logger != nil {
			stderrWriter := NewPrefixWriter(logger, true)
			_, _ = io.Copy(io.MultiWriter(&stderrBuf, stderrWriter), stderr)
			stderrWriter.Flush()
		} else {
			_, _ = io.Copy(io.MultiWriter(os.Stderr, &stderrBuf), stderr)
		}
	}()
	
	return stdoutBuf.String(), stderrBuf.String()
}

// getExitCode extracts the exit code from a process error.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	
	return -1
}

func execCommandWithOutputAndLogger(ctx context.Context, cmd string, logger *Logger) (int, string, string, error) {
	splitCmd, err := parseCommand(cmd)
	if err != nil {
		return -1, "", "", err
	}
	
	c := exec.CommandContext(ctx, splitCmd[0], splitCmd[1:]...) //nolint:gosec
	return executeCommandWithPipes(c, logger)
}

// executeCommandWithPipes handles command execution with pipes and output processing.
func executeCommandWithPipes(c *exec.Cmd, logger *Logger) (int, string, string, error) {
	stdout, stderr, err := setupCommandPipes(c)
	if err != nil {
		return -1, "", "", err
	}

	err = c.Start()
	if err != nil {
		return getExitCode(err), "", "", fmt.Errorf("command failed: %w", err)
	}

	return waitForCommandCompletion(c, stdout, stderr, logger)
}

// waitForCommandCompletion waits for command to finish and processes output.
func waitForCommandCompletion(c *exec.Cmd, stdout, stderr io.ReadCloser, logger *Logger) (int, string, string, error) {
	var wg sync.WaitGroup
	stdoutStr, stderrStr := handleCommandOutput(stdout, stderr, logger, &wg)
	
	err := c.Wait()
	_ = stderr.Close()
	_ = stdout.Close()
	wg.Wait()
	
	exitCode := getExitCode(err)
	if err != nil {
		return exitCode, stdoutStr, stderrStr, fmt.Errorf("command failed: %w", err)
	}
	
	return exitCode, stdoutStr, stderrStr, nil
}

// initializeLogging initializes logging for the retry execution.
func (r *Retry) initializeLogging(logger *Logger) {
	if logger == nil {
		return
	}
	
	maxTries := 0
	if mt, ok := r.condition.(*StopOnMaxTries); ok {
		if mt.maxTries <= ^uint(0)>>1 { // Check if fits in int
			maxTries = int(mt.maxTries)
		}
	}
	
	backoffName := "fixed"
	if r.backoff != nil {
		backoffName = "configured"
	}
	
	logger.StartExecution(r.cmd, maxTries, backoffName)
}

// executeRetryLoop runs the main retry loop logic.
func (r *Retry) executeRetryLoop(logger *Logger) error {
	var err error
	
	for r.shouldContinue() {
		if logger != nil {
			logger.StartAttempt(r.tries + 1)
		}

		err = r.executeSingleTryWithLogger(logger)
		success := err == nil

		if logger != nil {
			logger.EndAttempt(r.getLastExitCode(), success)
		}

		if success {
			break
		}

		delay := r.performBackoffWithDelay()
		if logger != nil && delay > 0 {
			logger.LogRetryDelay(delay)
		}
	}
	
	return err
}

// determineFailureReason determines why the retry execution stopped.
func (r *Retry) determineFailureReason() (string, string) {
	if r.condition.GetCtx().Err() != nil {
		return "context timeout", "timeout"
	} else if r.condition.IsLimitReached() {
		return "max tries reached", "max tries"
	}
	return "", ""
}
