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
	"syscall"
	"time"

	"github.com/go-andiamo/splitter"
	"github.com/sgaunet/retry/pkg/logger"
)

var (
	// ErrConditionNil is returned when the condition is nil.
	ErrConditionNil = errors.New("condition is nil")
	// ErrMaxTriesReached is returned when the maximum number of tries is reached.
	ErrMaxTriesReached = errors.New("max tries reached")
	// ErrEmptyCommand is returned when the command is empty.
	ErrEmptyCommand = errors.New("empty command")
	// ErrCommandTerminatedBySignal is returned when the command is terminated by signal.
	ErrCommandTerminatedBySignal = errors.New("command terminated by signal")
)

const (
	// outputStreams represents the number of output streams (stdout and stderr).
	outputStreams = 2
	// signalExitCodeBase is the base value for signal exit codes (128).
	signalExitCodeBase = 128
)

// Retry is a struct that represents a retry mechanism for executing commands.
type Retry struct {
	cmd               string
	tries             int
	condition         ConditionRetryer
	backoff           BackoffStrategy
	lastExitCode      int
	successConditions []ConditionRetryer
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

// SetSuccessConditions sets the success conditions to be evaluated separately from stop conditions.
func (r *Retry) SetSuccessConditions(conditions []ConditionRetryer) {
	r.successConditions = conditions
}

// GetSuccessConditions returns the success conditions for debugging.
func (r *Retry) GetSuccessConditions() []ConditionRetryer {
	return r.successConditions
}

// Run executes the command with retries based on the condition.
// It returns an error if the command fails or if the maximum number of tries is reached.
// It also logs the output of the command to the provided logger.
func (r *Retry) Run(_ *slog.Logger) error {
	return r.RunWithLogger(context.TODO(), nil)
}


// RunWithLogger executes the command with retry logic and logging support.
// The context parameter allows for cancellation and timeout control from the caller.
// The appLogger parameter provides logging for retry execution and debug traces.
//
//nolint:contextcheck // Context is properly used for cancellation
func (r *Retry) RunWithLogger(ctx context.Context, appLogger logger.Logger) error {
	// If no context is provided, use background context for backward compatibility
	if ctx == nil {
		ctx = context.Background()
	}

	if appLogger != nil {
		maxTries := r.extractMaxTriesFromCondition()
		backoffType := "none"
		if r.backoff != nil {
			backoffType = "configured"
		}
		appLogger.Debug("Retry loop starting", "command", r.cmd, "max_tries", maxTries, "backoff", backoffType, "success_conditions", len(r.successConditions))
	}

	err := r.executeRetryLoop(ctx, appLogger)

	if appLogger != nil {
		if err == nil {
			appLogger.Info("Retry execution completed successfully", "attempts", r.tries, "final_exit_code", r.lastExitCode)
		} else {
			// Determine failure reason
			var failureReason, stopCondition string
			if r.condition.GetCtx().Err() != nil {
				failureReason = "context timeout"
				stopCondition = "timeout"
			} else if r.condition.IsLimitReached() {
				failureReason = "max tries reached"
				stopCondition = "max tries"
			}
			appLogger.Warn("Retry execution failed", "reason", failureReason, "stop_condition", stopCondition, "attempts", r.tries, "final_exit_code", r.lastExitCode)
		}
	}

	return r.getFinalError(ctx, err)
}

// shouldContinue checks if the retry loop should continue.
func (r *Retry) shouldContinue(ctx context.Context, appLogger logger.Logger) bool {
	// Check if the root context (signal handling) is cancelled first
	if ctx.Err() != nil {
		if appLogger != nil {
			appLogger.Debug("Context cancelled, stopping retry loop", "error", ctx.Err())
		}
		return false
	}

	if r.condition.GetCtx().Err() != nil {
		if appLogger != nil {
			appLogger.Debug("Condition context cancelled, stopping retry loop", "error", r.condition.GetCtx().Err())
		}
		return false
	}

	if r.condition.IsLimitReached() {
		if appLogger != nil {
			// Provide detailed logging based on condition type
			switch cond := r.condition.(type) {
			case *StopOnMaxTries:
				appLogger.Debug("Stop condition reached: maximum tries exceeded", "attempts", r.tries, "max_tries", cond.maxTries)
			case *StopOnMaxExecutionTime:
				appLogger.Debug("Stop condition reached: maximum execution time exceeded", "attempts", r.tries)
			case *StopOnExitCode:
				appLogger.Debug("Stop condition reached: exit code matched", "attempts", r.tries, "exit_code", r.lastExitCode)
			case *StopOnTimeout:
				appLogger.Debug("Stop condition reached: timeout", "attempts", r.tries)
			case *CompositeCondition:
				appLogger.Debug("Stop condition reached: composite condition met", "attempts", r.tries)
			default:
				appLogger.Debug("Stop condition reached", "attempts", r.tries)
			}
		}
		return false
	}

	return true
}


// getFinalError determines the final error to return.
func (r *Retry) getFinalError(ctx context.Context, err error) error {
	// Check root context first (signal handling)
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}
	
	if r.condition.GetCtx().Err() != nil {
		return fmt.Errorf("context error: %w", r.condition.GetCtx().Err())
	}
	
	// If success conditions were met, don't return max tries error
	if r.isSuccessConditionMet() {
		return nil
	}
	
	if r.condition.IsLimitReached() && err != nil {
		return ErrMaxTriesReached
	}
	return err
}


// performBackoffWithDelay handles the delay and returns the delay duration.
func (r *Retry) performBackoffWithDelay(appLogger logger.Logger) time.Duration {
	if r.backoff != nil {
		delay := r.backoff.NextDelay(r.tries)
		if delay > 0 {
			if appLogger != nil {
				// Log detailed backoff information based on strategy type
				switch b := r.backoff.(type) {
				case *FixedBackoff:
					appLogger.Debug("Applying fixed backoff delay", "delay", delay, "attempt", r.tries, "strategy", "fixed")
				case *ExponentialBackoff:
					appLogger.Debug("Applying exponential backoff delay", "delay", delay, "attempt", r.tries, "strategy", "exponential", "base_delay", b.BaseDelay, "max_delay", b.MaxDelay, "multiplier", b.Multiplier)
				case *LinearBackoff:
					appLogger.Debug("Applying linear backoff delay", "delay", delay, "attempt", r.tries, "strategy", "linear")
				case *FibonacciBackoff:
					appLogger.Debug("Applying fibonacci backoff delay", "delay", delay, "attempt", r.tries, "strategy", "fibonacci")
				case *JitterBackoff:
					appLogger.Debug("Applying jitter backoff delay (with randomization)", "delay", delay, "attempt", r.tries, "strategy", "jitter")
				default:
					appLogger.Debug("Applying custom backoff delay", "delay", delay, "attempt", r.tries, "strategy", "custom")
				}
			}
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

// executeSingleTryWithLogger executes a single retry attempt with logging.
func (r *Retry) executeSingleTryWithLogger(ctx context.Context, appLogger logger.Logger) error {
	if r.condition != nil {
		r.condition.StartTry()
	}

	// Start try for success conditions
	for _, successCond := range r.successConditions {
		successCond.StartTry()
	}
	r.tries++

	if appLogger != nil {
		appLogger.Debug("Executing command", "attempt", r.tries, "command", r.cmd)
	}

	startTime := time.Now()
	rc, stdout, stderr, err := execCommandWithOutputAndLogger(ctx, r.cmd, appLogger)
	duration := time.Since(startTime)
	r.lastExitCode = rc

	if appLogger != nil {
		appLogger.Debug("Command completed", "attempt", r.tries, "exit_code", rc, "duration", duration, "error", err != nil)
	}

	// Pass exit code and output to enhanced conditions
	if enhanced, ok := r.condition.(EnhancedConditionRetryer); ok {
		enhanced.SetLastExitCode(rc)
		enhanced.SetLastOutput(stdout, stderr)
	}

	// Pass exit code and output to success conditions
	for _, successCond := range r.successConditions {
		if enhanced, ok := successCond.(EnhancedConditionRetryer); ok {
			enhanced.SetLastExitCode(rc)
			enhanced.SetLastOutput(stdout, stderr)
		}
	}

	if r.condition != nil {
		r.condition.EndTry()
	}

	// End try for success conditions
	for _, successCond := range r.successConditions {
		successCond.EndTry()
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

// checkSignalTermination checks if the process was terminated by a signal and returns appropriate exit code.
func checkSignalTermination(c *exec.Cmd, err error) (int, error) {
	if c.ProcessState == nil || c.ProcessState.Success() {
		return 0, nil
	}
	
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return 0, nil
	}
	
	status, ok := exitError.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return 0, nil
	}
	
	// Process was terminated by signal, return appropriate exit code
	signalExitCode := signalExitCodeBase + int(status.Signal())
	signalErr := fmt.Errorf("%w: %v", ErrCommandTerminatedBySignal, status.Signal())
	return signalExitCode, signalErr
}

func execCommandWithOutputAndLogger(ctx context.Context, cmd string, appLogger logger.Logger) (int, string, string, error) {
	splitCmd, err := parseCommand(cmd)
	if err != nil {
		if appLogger != nil {
			appLogger.Debug("Failed to parse command", "command", cmd, "error", err)
		}
		return -1, "", "", err
	}

	if appLogger != nil {
		appLogger.Debug("Parsed command", "executable", splitCmd[0], "args", splitCmd[1:])
	}

	c := exec.CommandContext(ctx, splitCmd[0], splitCmd[1:]...) //nolint:gosec

	// Set up process group for better signal handling
	c.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return executeCommandWithPipes(c)
}

// executeCommandWithPipes handles command execution with pipes and output processing.
func executeCommandWithPipes(c *exec.Cmd) (int, string, string, error) {
	stdout, stderr, err := setupCommandPipes(c)
	if err != nil {
		return -1, "", "", err
	}

	err = c.Start()
	if err != nil {
		return getExitCode(err), "", "", fmt.Errorf("command failed: %w", err)
	}

	return waitForCommandCompletion(c, stdout, stderr)
}

// waitForCommandCompletion waits for command to finish and processes output.
func waitForCommandCompletion(c *exec.Cmd, stdout, stderr io.ReadCloser) (int, string, string, error) {
	var wg sync.WaitGroup
	var stdoutBuf, stderrBuf strings.Builder

	wg.Add(outputStreams)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(os.Stdout, &stdoutBuf), stdout)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(os.Stderr, &stderrBuf), stderr)
	}()
	
	// Wait for command to complete
	// The context cancellation will automatically terminate the process
	// since we used exec.CommandContext
	err := c.Wait()
	_ = stderr.Close()
	_ = stdout.Close()
	wg.Wait()
	
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()
	
	exitCode := getExitCode(err)
	
	// Check if the process was terminated by a signal
	if signalExitCode, signalErr := checkSignalTermination(c, err); signalExitCode != 0 {
		return signalExitCode, stdoutStr, stderrStr, signalErr
	}
	
	if err != nil {
		return exitCode, stdoutStr, stderrStr, fmt.Errorf("command failed: %w", err)
	}
	
	return exitCode, stdoutStr, stderrStr, nil
}

// extractMaxTriesFromCondition extracts the maxTries value from the condition.
func (r *Retry) extractMaxTriesFromCondition() int {
	if mt, ok := r.condition.(*StopOnMaxTries); ok {
		if mt.maxTries <= ^uint(0)>>1 { // Check if fits in int
			return int(mt.maxTries)
		}
		return 0
	}
	
	if comp, ok := r.condition.(*CompositeCondition); ok {
		// For composite conditions, look for StopOnMaxTries within the composite
		for _, cond := range comp.GetConditions() {
			if mt, ok := cond.(*StopOnMaxTries); ok {
				if mt.maxTries <= ^uint(0)>>1 { // Check if fits in int
					return int(mt.maxTries)
				}
			}
		}
	}
	
	return 0
}

// executeRetryLoop runs the main retry loop logic.
func (r *Retry) executeRetryLoop(ctx context.Context, appLogger logger.Logger) error {
	var err error
	maxTries := r.extractMaxTriesFromCondition()

	for r.shouldContinue(ctx, appLogger) {
		attemptNum := r.tries + 1
		if appLogger != nil {
			appLogger.Info("Attempting command", "attempt", attemptNum, "max_tries", maxTries)
		}

		err = r.executeSingleTryWithLogger(ctx, appLogger)

		// Check if this is a success condition (even if err != nil)
		// Success conditions that have IsLimitReached() == true mean success was achieved
		successCondMet := r.isSuccessConditionMet()
		success := err == nil || successCondMet

		if appLogger != nil {
			if successCondMet {
				appLogger.Info("Command succeeded with success condition met", "attempt", r.tries, "exit_code", r.getLastExitCode())
				r.logSuccessConditionDetails(appLogger)
			} else if err == nil {
				appLogger.Info("Command succeeded", "attempt", r.tries, "exit_code", 0)
			} else {
				appLogger.Warn("Command failed", "attempt", r.tries, "exit_code", r.getLastExitCode())
			}
		}

		if success {
			// Clear the error if success condition was met
			if successCondMet {
				err = nil
			}
			break
		}

		delay := r.performBackoffWithDelay(appLogger)
		if appLogger != nil && delay > 0 {
			appLogger.Info("Waiting before retry", "delay", delay)
		}
	}

	return err
}

// isSuccessConditionMet checks if any success condition has been met.
func (r *Retry) isSuccessConditionMet() bool {
	// Check dedicated success conditions first
	for _, cond := range r.successConditions {
		if cond.IsLimitReached() {
			return true
		}
	}
	
	// Fallback: check if the main condition is a success-type condition
	if r.condition == nil {
		return false
	}
	
	// Check if this is a SuccessOnExitCode, SuccessContains, or SuccessRegex
	switch cond := r.condition.(type) {
	case *SuccessOnExitCode:
		return cond.IsLimitReached()
	case *SuccessContains:
		return cond.IsLimitReached()
	case *SuccessRegex:
		return cond.IsLimitReached()
	case *CompositeCondition:
		// For composite conditions, check each sub-condition
		return r.checkCompositeForSuccess(cond)
	}
	
	return false
}

// checkCompositeForSuccess checks if any success condition in a composite has been met.
func (r *Retry) checkCompositeForSuccess(comp *CompositeCondition) bool {
	// Check each condition in the composite
	for _, cond := range comp.GetConditions() {
		switch c := cond.(type) {
		case *SuccessOnExitCode:
			if c.IsLimitReached() {
				return true
			}
		case *SuccessContains:
			if c.IsLimitReached() {
				return true
			}
		case *SuccessRegex:
			if c.IsLimitReached() {
				return true
			}
		}
	}
	return false
}

// logSuccessConditionDetails logs detailed information about which success condition was met.
func (r *Retry) logSuccessConditionDetails(appLogger logger.Logger) {
	if appLogger == nil {
		return
	}

	// Log dedicated success conditions
	for i, cond := range r.successConditions {
		if !cond.IsLimitReached() {
			continue
		}

		switch c := cond.(type) {
		case *SuccessOnExitCode:
			appLogger.Debug("Success condition details: exit code matched", "condition_index", i, "exit_code", c.lastExitCode, "expected_codes", c.successCodes)
		case *SuccessContains:
			appLogger.Debug("Success condition details: output pattern found", "condition_index", i, "pattern", c.pattern)
		case *SuccessRegex:
			appLogger.Debug("Success condition details: regex matched", "condition_index", i, "pattern", c.pattern)
		}
	}

	// Also check main condition if it's a success type
	switch cond := r.condition.(type) {
	case *SuccessOnExitCode:
		if cond.IsLimitReached() {
			appLogger.Debug("Main success condition met: exit code", "exit_code", cond.lastExitCode, "expected_codes", cond.successCodes)
		}
	case *SuccessContains:
		if cond.IsLimitReached() {
			appLogger.Debug("Main success condition met: pattern found", "pattern", cond.pattern)
		}
	case *SuccessRegex:
		if cond.IsLimitReached() {
			appLogger.Debug("Main success condition met: regex matched", "pattern", cond.pattern)
		}
	}
}

