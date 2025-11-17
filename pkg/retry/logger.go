package retry

import (
	"encoding/json"
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
	// LogLevelError shows only error messages.
	LogLevelError LogLevel = iota
	// LogLevelWarn shows warning and error messages.
	LogLevelWarn
	// LogLevelInfo shows informational, warning, and error messages.
	LogLevelInfo
	// LogLevelDebug shows all messages including debug information.
	LogLevelDebug
	// LogLevelQuiet shows minimal output.
	LogLevelQuiet
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
	// OutputModeJSON outputs structured JSON data.
	OutputModeJSON
)

// Logger provides enhanced logging with colors and better formatting.
type Logger struct {
	out        io.Writer
	err        io.Writer
	level      LogLevel
	mode       OutputMode
	noColor    bool
	startTime  time.Time
	logFile    io.WriteCloser // Optional log file
	
	// Color functions
	dimColor     func(a ...any) string
	successColor func(a ...any) string
	errorColor   func(a ...any) string
	warnColor    func(a ...any) string
	boldColor    func(a ...any) string
	
	// State tracking
	currentAttempt int
	maxAttempts    int
	lastExitCode   int
	summary        *ExecutionSummary
	
	// JSON output tracking
	jsonOutput *JSONOutput
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

// JSONAttempt represents a single retry attempt in JSON output.
type JSONAttempt struct {
	Attempt   int           `json:"attempt"`
	ExitCode  int           `json:"exit_code"`
	Duration  string        `json:"duration"`
	Output    string        `json:"output"`
	Success   bool          `json:"success"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// JSONOutput represents the complete JSON output structure.
type JSONOutput struct {
	Command        string         `json:"command"`
	TotalAttempts  int           `json:"total_attempts"`
	Successful     bool          `json:"successful"`
	TotalDuration  string        `json:"total_duration"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	FinalExitCode  int           `json:"final_exit_code"`
	BackoffStrategy string       `json:"backoff_strategy"`
	MaxAttempts    int           `json:"max_attempts"`
	FailureReason  string        `json:"failure_reason,omitempty"`
	StopCondition  string        `json:"stop_condition,omitempty"`
	Attempts       []JSONAttempt `json:"attempts"`
}

// NewLogger creates a new enhanced logger.
func NewLogger(level LogLevel, mode OutputMode, noColor bool) *Logger {
	return NewLoggerWithFile(level, mode, noColor, "")
}

// NewLoggerWithFile creates a new enhanced logger with optional file logging.
func NewLoggerWithFile(level LogLevel, mode OutputMode, noColor bool, logFilePath string) *Logger {
	l := &Logger{
		out:       os.Stdout,
		err:       os.Stderr,
		level:     level,
		mode:      mode,
		noColor:   noColor,
		startTime: time.Now(),
		summary:   &ExecutionSummary{StartTime: time.Now()},
	}
	
	// Initialize JSON output if needed
	if mode == OutputModeJSON {
		l.jsonOutput = &JSONOutput{
			StartTime: time.Now(),
			Attempts:  make([]JSONAttempt, 0),
		}
	}
	
	// Setup log file if specified
	if logFilePath != "" {
		// Basic validation to prevent directory traversal
		if !strings.Contains(logFilePath, "..") {
			file, err := os.Create(logFilePath) // #nosec G304 - user-provided log file path is intentional
			if err == nil {
				l.logFile = file
			}
		}
	}
	
	l.setupColors()
	return l
}

// Close closes any open log file.
func (l *Logger) Close() error {
	if l.logFile != nil {
		if err := l.logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}
	return nil
}


// StartExecution begins tracking a new retry execution.
func (l *Logger) StartExecution(command string, maxAttempts int, backoffStrategy string) {
	l.summary.Command = command
	l.summary.MaxAttempts = maxAttempts
	l.summary.BackoffStrategy = backoffStrategy
	l.maxAttempts = maxAttempts
	
	// Initialize JSON output if needed
	if l.mode == OutputModeJSON && l.jsonOutput != nil {
		l.jsonOutput.Command = command
		l.jsonOutput.MaxAttempts = maxAttempts
		l.jsonOutput.BackoffStrategy = backoffStrategy
		l.jsonOutput.StartTime = time.Now()
	}
}

// StartAttempt logs the start of a new retry attempt.
func (l *Logger) StartAttempt(attempt int) {
	l.currentAttempt = attempt

	// Track JSON attempt start
	if l.mode == OutputModeJSON && l.jsonOutput != nil {
		jsonAttempt := JSONAttempt{
			Attempt:   attempt,
			StartTime: time.Now(),
		}
		l.jsonOutput.Attempts = append(l.jsonOutput.Attempts, jsonAttempt)
	}

	// Skip output for summary-only, JSON, or quiet modes (but not quiet-retries)
	if l.mode == OutputModeSummaryOnly || l.mode == OutputModeJSON {
		return
	}
	if l.level == LogLevelQuiet && l.mode != OutputModeQuietRetries {
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
		l.writeToLogFile(msg)
	} else {
		msg := fmt.Sprintf("%s %s", l.boldColor(attemptMsg), "Retrying...")
		_, _ = fmt.Fprintln(l.out, msg)
		l.writeToLogFile(msg)
	}
}

// LogCommandOutput logs output from the executed command with proper formatting.
func (l *Logger) LogCommandOutput(line string, isStderr bool) {
	l.storeJSONOutput(line)
	l.writeFileOutput(line, isStderr)
	
	if l.shouldSkipConsoleOutput() {
		return
	}
	
	l.writeConsoleOutput(line, isStderr)
}

// EndAttempt logs the result of an attempt.
func (l *Logger) EndAttempt(exitCode int, success bool) {
	l.lastExitCode = exitCode
	l.updateJSONAttemptData(exitCode, success)

	if l.shouldSkipAttemptOutput() {
		return
	}

	statusMsg := l.formatStatusMessage(exitCode, success)
	_, _ = fmt.Fprintln(l.out, statusMsg)
	l.writeToLogFile(statusMsg)

	if !success && l.currentAttempt < l.maxAttempts {
		_, _ = fmt.Fprintln(l.out) // Add blank line between attempts
	}
}

// LogRetryDelay logs information about retry delay.
func (l *Logger) LogRetryDelay(delay time.Duration) {
	if l.mode == OutputModeSummaryOnly || l.mode == OutputModeJSON || l.level == LogLevelQuiet {
		return
	}
	
	if delay > 0 {
		msg := l.dimColor(fmt.Sprintf("Waiting %v before retry...", delay))
		_, _ = fmt.Fprintln(l.out, msg)
		l.writeToLogFile(msg)
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
	
	if l.mode == OutputModeJSON {
		l.outputJSON(success, failureReason, stopCondition)
	} else {
		l.printSummary()
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	if l.level < LogLevelDebug || l.mode == OutputModeSummaryOnly || l.mode == OutputModeJSON {
		return
	}
	debugMsg := l.dimColor("DEBUG: " + msg)
	_, _ = fmt.Fprintln(l.out, debugMsg)
	l.writeToLogFile(debugMsg)
}

// Info logs an informational message.
func (l *Logger) Info(msg string) {
	if l.level < LogLevelInfo || l.mode == OutputModeSummaryOnly || l.mode == OutputModeJSON {
		return
	}
	_, _ = fmt.Fprintln(l.out, msg)
	l.writeToLogFile(msg)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	if l.level < LogLevelWarn || l.mode == OutputModeJSON {
		return
	}
	warnMsg := l.warnColor("WARN: " + msg)
	_, _ = fmt.Fprintln(l.err, warnMsg)
	l.writeToLogFile(warnMsg)
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	if l.level < LogLevelError || l.mode == OutputModeJSON {
		return
	}
	errorMsg := l.errorColor("ERROR: " + msg)
	_, _ = fmt.Fprintln(l.err, errorMsg)
	l.writeToLogFile(errorMsg)
}

// Verbose logs a verbose message (for backward compatibility).
func (l *Logger) Verbose(msg string) {
	if l.level != LogLevelVerbose || l.mode == OutputModeSummaryOnly || l.mode == OutputModeJSON {
		return
	}
	verboseMsg := l.dimColor(msg)
	_, _ = fmt.Fprintln(l.out, verboseMsg)
	l.writeToLogFile(verboseMsg)
}

// updateJSONAttemptData updates the JSON attempt data with exit code and success status.
func (l *Logger) updateJSONAttemptData(exitCode int, success bool) {
	if l.mode != OutputModeJSON || l.jsonOutput == nil || len(l.jsonOutput.Attempts) == 0 {
		return
	}

	lastAttemptIdx := len(l.jsonOutput.Attempts) - 1
	attempt := &l.jsonOutput.Attempts[lastAttemptIdx]
	attempt.ExitCode = exitCode
	attempt.Success = success
	attempt.EndTime = time.Now()
	attempt.Duration = attempt.EndTime.Sub(attempt.StartTime).String()
}

// shouldSkipAttemptOutput determines if attempt output should be skipped.
func (l *Logger) shouldSkipAttemptOutput() bool {
	if l.mode == OutputModeSummaryOnly || l.mode == OutputModeJSON {
		return true
	}
	return l.level == LogLevelQuiet && l.mode != OutputModeQuietRetries
}

// formatStatusMessage creates the status message for an attempt.
func (l *Logger) formatStatusMessage(exitCode int, success bool) string {
	if success {
		return l.successColor("✓ Success")
	}
	return l.errorColor(fmt.Sprintf("✗ Failed with exit code %d", exitCode))
}

// writeToLogFile writes a message to the log file if it exists.
func (l *Logger) writeToLogFile(msg string) {
	if l.logFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		_, _ = fmt.Fprintf(l.logFile, "[%s] %s\n", timestamp, msg)
	}
}

// storeJSONOutput stores command output for JSON mode.
func (l *Logger) storeJSONOutput(line string) {
	if l.mode != OutputModeJSON || l.jsonOutput == nil || len(l.jsonOutput.Attempts) == 0 {
		return
	}
	
	lastAttemptIdx := len(l.jsonOutput.Attempts) - 1
	if l.jsonOutput.Attempts[lastAttemptIdx].Output == "" {
		l.jsonOutput.Attempts[lastAttemptIdx].Output = line
	} else {
		l.jsonOutput.Attempts[lastAttemptIdx].Output += "\n" + line
	}
}

// writeFileOutput writes command output to log file if configured.
func (l *Logger) writeFileOutput(line string, isStderr bool) {
	if l.logFile == nil {
		return
	}
	
	prefix := "[STDOUT] "
	if isStderr {
		prefix = "[STDERR] "
	}
	l.writeToLogFile(prefix + line)
}

// shouldSkipConsoleOutput determines if console output should be skipped.
func (l *Logger) shouldSkipConsoleOutput() bool {
	if l.mode == OutputModeSummaryOnly || l.mode == OutputModeJSON {
		return true
	}

	// In quiet mode (but not quiet-retries), suppress all command output
	if l.level == LogLevelQuiet && l.mode != OutputModeQuietRetries {
		return true
	}

	return l.mode == OutputModeQuietRetries && l.currentAttempt < l.maxAttempts
}

// writeConsoleOutput writes formatted output to console.
func (l *Logger) writeConsoleOutput(line string, isStderr bool) {
	prefix := l.dimColor("│ ")
	output := line
	if isStderr {
		output = l.errorColor(line)
	}
	
	_, _ = fmt.Fprintf(l.out, "%s%s\n", prefix, output)
}

// outputJSON outputs the execution result as JSON.
func (l *Logger) outputJSON(success bool, failureReason string, stopCondition string) {
	if l.jsonOutput == nil {
		return
	}
	
	l.jsonOutput.EndTime = time.Now()
	l.jsonOutput.TotalDuration = l.jsonOutput.EndTime.Sub(l.jsonOutput.StartTime).String()
	l.jsonOutput.TotalAttempts = l.currentAttempt
	l.jsonOutput.Successful = success
	l.jsonOutput.FinalExitCode = l.lastExitCode
	l.jsonOutput.FailureReason = failureReason
	l.jsonOutput.StopCondition = stopCondition
	
	jsonData, err := json.MarshalIndent(l.jsonOutput, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(l.err, "Error marshaling JSON: %v\n", err)
		return
	}
	
	_, _ = fmt.Fprintln(l.out, string(jsonData))
	
	// Also write to log file if available
	if l.logFile != nil {
		l.writeToLogFile("JSON OUTPUT:")
		l.writeToLogFile(string(jsonData))
	}
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
	// Don't print summary for summary-only mode (it's handled differently)
	// But DO print summary for quiet mode (that's the "final result")
	if l.mode == OutputModeSummaryOnly {
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