package retry

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

// LogLevel represents different log levels.
type LogLevel int

const (
	// LogLevelQuiet shows minimal output.
	LogLevelQuiet LogLevel = iota
	// LogLevelNormal shows standard output.
	LogLevelNormal
	// LogLevelVerbose shows detailed output.
	LogLevelVerbose
)

const (
	// summaryHeaderLength defines the length of summary header separators.
	summaryHeaderLength = 15
	// summaryTotalLength defines the total length of the summary footer.
	summaryTotalLength = 41
)

// OutputMode represents different output formatting modes.
type OutputMode int

const (
	// OutputModeNormal shows all output with full formatting.
	OutputModeNormal OutputMode = iota
	// OutputModeQuietRetries only shows output for the final attempt.
	OutputModeQuietRetries
	// OutputModeSummaryOnly shows only the execution summary.
	OutputModeSummaryOnly
)

// Logger provides enhanced logging with colors and better formatting.
type Logger struct {
	out        io.Writer
	err        io.Writer
	level      LogLevel
	mode       OutputMode
	noColor    bool
	startTime  time.Time
	
	// Color functions
	dimColor     func(a ...interface{}) string
	successColor func(a ...interface{}) string
	errorColor   func(a ...interface{}) string
	warnColor    func(a ...interface{}) string
	boldColor    func(a ...interface{}) string
	
	// State tracking
	currentAttempt int
	maxAttempts    int
	lastExitCode   int
	summary        *ExecutionSummary
}

// ExecutionSummary holds information about the retry execution.
type ExecutionSummary struct {
	Command           string
	StartTime         time.Time
	EndTime           time.Time
	TotalAttempts     int
	MaxAttempts       int
	FinalExitCode     int
	Success           bool
	FailureReason     string
	StopCondition     string
	TotalDuration     time.Duration
	BackoffStrategy   string
}

// NewLogger creates a new enhanced logger.
func NewLogger(level LogLevel, mode OutputMode, noColor bool) *Logger {
	l := &Logger{
		out:       os.Stdout,
		err:       os.Stderr,
		level:     level,
		mode:      mode,
		noColor:   noColor,
		startTime: time.Now(),
		summary:   &ExecutionSummary{StartTime: time.Now()},
	}
	
	l.setupColors()
	return l
}


// StartExecution begins tracking a new retry execution.
func (l *Logger) StartExecution(command string, maxAttempts int, backoffStrategy string) {
	l.summary.Command = command
	l.summary.MaxAttempts = maxAttempts
	l.summary.BackoffStrategy = backoffStrategy
	l.maxAttempts = maxAttempts
}

// StartAttempt logs the start of a new retry attempt.
func (l *Logger) StartAttempt(attempt int) {
	l.currentAttempt = attempt
	
	if l.mode == OutputModeSummaryOnly {
		return
	}
	
	var maxStr string
	if l.maxAttempts > 0 {
		maxStr = fmt.Sprintf("/%d", l.maxAttempts)
	} else {
		maxStr = ""
	}
	
	attemptMsg := fmt.Sprintf("[%d%s]", attempt, maxStr)
	
	if attempt == 1 {
		msg := fmt.Sprintf("%s %s", l.boldColor(attemptMsg), "Attempting command...")
		_, _ = fmt.Fprintln(l.out, msg)
	} else {
		msg := fmt.Sprintf("%s %s", l.boldColor(attemptMsg), "Retrying...")
		_, _ = fmt.Fprintln(l.out, msg)
	}
}

// LogCommandOutput logs output from the executed command with proper formatting.
func (l *Logger) LogCommandOutput(line string, isStderr bool) {
	if l.mode == OutputModeSummaryOnly {
		return
	}
	
	if l.mode == OutputModeQuietRetries && l.currentAttempt < l.maxAttempts {
		// Skip output for non-final attempts in quiet mode
		return
	}
	
	prefix := l.dimColor("│ ")
	var output string
	
	if isStderr {
		output = l.errorColor(line)
	} else {
		output = line
	}
	
	_, _ = fmt.Fprintf(l.out, "%s%s\n", prefix, output)
}

// EndAttempt logs the result of an attempt.
func (l *Logger) EndAttempt(exitCode int, success bool) {
	l.lastExitCode = exitCode
	
	if l.mode == OutputModeSummaryOnly {
		return
	}
	
	var statusMsg string
	if success {
		statusMsg = l.successColor("✓ Success")
	} else {
		statusMsg = l.errorColor(fmt.Sprintf("✗ Failed with exit code %d", exitCode))
	}
	
	_, _ = fmt.Fprintln(l.out, statusMsg)
	
	if !success && l.currentAttempt < l.maxAttempts {
		_, _ = fmt.Fprintln(l.out) // Add blank line between attempts
	}
}

// LogRetryDelay logs information about retry delay.
func (l *Logger) LogRetryDelay(delay time.Duration) {
	if l.mode == OutputModeSummaryOnly || l.level == LogLevelQuiet {
		return
	}
	
	if delay > 0 {
		msg := l.dimColor(fmt.Sprintf("Waiting %v before retry...", delay))
		_, _ = fmt.Fprintln(l.out, msg)
	}
}

// EndExecution finalizes the execution and logs the summary.
func (l *Logger) EndExecution(success bool, failureReason string, stopCondition string) {
	l.summary.EndTime = time.Now()
	l.summary.TotalDuration = l.summary.EndTime.Sub(l.summary.StartTime)
	l.summary.TotalAttempts = l.currentAttempt
	l.summary.FinalExitCode = l.lastExitCode
	l.summary.Success = success
	l.summary.FailureReason = failureReason
	l.summary.StopCondition = stopCondition
	
	l.printSummary()
}


// Info logs an informational message.
func (l *Logger) Info(msg string) {
	if l.level == LogLevelQuiet || l.mode == OutputModeSummaryOnly {
		return
	}
	_, _ = fmt.Fprintln(l.out, msg)
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	if l.level == LogLevelQuiet {
		return
	}
	_, _ = fmt.Fprintln(l.err, l.errorColor(msg))
}

// Verbose logs a verbose message.
func (l *Logger) Verbose(msg string) {
	if l.level != LogLevelVerbose || l.mode == OutputModeSummaryOnly {
		return
	}
	_, _ = fmt.Fprintln(l.out, l.dimColor(msg))
}

// setupColors initializes color functions based on noColor setting.
func (l *Logger) setupColors() {
	if l.noColor {
		l.dimColor = color.New().SprintFunc()
		l.successColor = color.New().SprintFunc()
		l.errorColor = color.New().SprintFunc()
		l.warnColor = color.New().SprintFunc()
		l.boldColor = color.New().SprintFunc()
	} else {
		l.dimColor = color.New(color.FgHiBlack).SprintFunc()
		l.successColor = color.New(color.FgGreen).SprintFunc()
		l.errorColor = color.New(color.FgRed).SprintFunc()
		l.warnColor = color.New(color.FgYellow).SprintFunc()
		l.boldColor = color.New(color.Bold).SprintFunc()
	}
}

// printSummary prints the final execution summary.
func (l *Logger) printSummary() {
	if l.level == LogLevelQuiet {
		return
	}
	
	_, _ = fmt.Fprintln(l.out)
	
	// Summary header
	headerLine := strings.Repeat("═", summaryHeaderLength)
	header := fmt.Sprintf("%s SUMMARY %s", headerLine, headerLine)
	_, _ = fmt.Fprintln(l.out, l.boldColor(header))
	
	// Result
	var resultMsg string
	if l.summary.Success {
		resultMsg = l.successColor("Success")
	} else {
		reason := l.summary.FailureReason
		if reason == "" {
			reason = "Command failed"
		}
		resultMsg = l.errorColor(fmt.Sprintf("Failed (%s)", reason))
	}
	_, _ = fmt.Fprintf(l.out, "Result: %s\n", resultMsg)
	
	// Attempts
	var attemptsStr string
	if l.summary.MaxAttempts > 0 {
		attemptsStr = fmt.Sprintf("%d/%d", l.summary.TotalAttempts, l.summary.MaxAttempts)
	} else {
		attemptsStr = strconv.Itoa(l.summary.TotalAttempts)
	}
	_, _ = fmt.Fprintf(l.out, "Attempts: %s\n", attemptsStr)
	
	// Duration
	_, _ = fmt.Fprintf(l.out, "Duration: %v\n", l.summary.TotalDuration.Round(time.Millisecond))
	
	// Final exit code
	_, _ = fmt.Fprintf(l.out, "Final exit code: %d\n", l.summary.FinalExitCode)
	
	// Stop condition (if applicable)
	if l.summary.StopCondition != "" {
		_, _ = fmt.Fprintf(l.out, "Stop condition: %s\n", l.summary.StopCondition)
	}
	
	// Backoff strategy (if verbose)
	if l.level == LogLevelVerbose && l.summary.BackoffStrategy != "" {
		_, _ = fmt.Fprintf(l.out, "Backoff strategy: %s\n", l.summary.BackoffStrategy)
	}
	
	// Command (if verbose)
	if l.level == LogLevelVerbose {
		_, _ = fmt.Fprintf(l.out, "Command: %s\n", l.summary.Command)
	}
	
	_, _ = fmt.Fprintln(l.out, strings.Repeat("═", summaryTotalLength))
}