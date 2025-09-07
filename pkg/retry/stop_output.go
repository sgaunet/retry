package retry

import (
	"context"
	"regexp"
	"strings"
)

// StopOnOutputPattern stops retrying based on output pattern matching.
type StopOnOutputPattern struct {
	pattern      string
	regex        *regexp.Regexp
	contains     bool // true for "contains", false for "not contains"
	shouldStop   bool
	lastStdout   string
	lastStderr   string
}

// NewStopOnOutputContains creates a condition that stops when output contains the pattern.
func NewStopOnOutputContains(pattern string) (*StopOnOutputPattern, error) {
	// Try to compile as regex, fall back to simple string matching
	var regex *regexp.Regexp
	regex, _ = regexp.Compile(pattern) // Ignore error, will use string matching if invalid
	
	return &StopOnOutputPattern{
		pattern:    pattern,
		regex:      regex,
		contains:   true,
		shouldStop: false,
	}, nil
}

// NewStopOnOutputNotContains creates a condition that stops when output doesn't contain the pattern.
func NewStopOnOutputNotContains(pattern string) (*StopOnOutputPattern, error) {
	// Try to compile as regex, fall back to simple string matching
	var regex *regexp.Regexp
	regex, _ = regexp.Compile(pattern) // Ignore error, will use string matching if invalid
	
	return &StopOnOutputPattern{
		pattern:    pattern,
		regex:      regex,
		contains:   false,
		shouldStop: false,
	}, nil
}

// GetCtx returns a background context as pattern matching doesn't need timeout.
func (s *StopOnOutputPattern) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if we should stop based on output pattern.
func (s *StopOnOutputPattern) IsLimitReached() bool {
	return s.shouldStop
}

// StartTry does nothing for output pattern condition.
func (s *StopOnOutputPattern) StartTry() {}

// EndTry does nothing for output pattern condition.
func (s *StopOnOutputPattern) EndTry() {}

// SetLastExitCode is not used by output pattern condition.
func (s *StopOnOutputPattern) SetLastExitCode(_ int) {
	// Not used by output pattern condition
}

// SetLastOutput updates the last output and checks if we should stop.
func (s *StopOnOutputPattern) SetLastOutput(stdout, stderr string) {
	s.lastStdout = stdout
	s.lastStderr = stderr
	
	// Combine stdout and stderr for pattern matching
	combined := stdout + stderr
	
	// Check if pattern matches
	var matches bool
	if s.regex != nil {
		matches = s.regex.MatchString(combined)
	} else {
		// Fall back to simple string contains
		matches = strings.Contains(combined, s.pattern)
	}
	
	// Determine if we should stop
	if s.contains {
		// Stop when pattern IS found
		s.shouldStop = matches
	} else {
		// Stop when pattern is NOT found
		s.shouldStop = !matches
	}
}