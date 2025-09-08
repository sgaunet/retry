package retry

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// RetryOnExitCode implements retry logic based on specific exit codes.
// It will only retry if the exit code matches one of the specified codes.
//
//nolint:revive // Prefix is meaningful to distinguish from StopOnExitCode
type RetryOnExitCode struct {
	retryCodes   []int
	lastExitCode int
	shouldRetry  bool
}

// NewRetryOnExitCode creates a new retry condition based on exit codes.
func NewRetryOnExitCode(codes []int) *RetryOnExitCode {
	return &RetryOnExitCode{
		retryCodes:   codes,
		lastExitCode: -1,
		shouldRetry:  true, // Initially allow retry
	}
}

// GetCtx returns a background context.
func (r *RetryOnExitCode) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if we should stop retrying.
func (r *RetryOnExitCode) IsLimitReached() bool {
	return !r.shouldRetry
}

// StartTry does nothing for retry exit code condition.
func (r *RetryOnExitCode) StartTry() {}

// EndTry does nothing for retry exit code condition.
func (r *RetryOnExitCode) EndTry() {}

// SetLastExitCode updates the last exit code and determines if we should retry.
func (r *RetryOnExitCode) SetLastExitCode(code int) {
	r.lastExitCode = code
	r.shouldRetry = false
	
	// Check if this exit code is in our retry list
	for _, retryCode := range r.retryCodes {
		if code == retryCode {
			r.shouldRetry = true
			return
		}
	}
}

// SetLastOutput is not used by retry exit code condition.
func (r *RetryOnExitCode) SetLastOutput(_, _ string) {}

// RetryIfContains implements retry logic based on output pattern matching.
// It will retry if the output contains a specific pattern.
//
//nolint:revive // Prefix is meaningful to distinguish from stop conditions
type RetryIfContains struct {
	pattern    string
	regex      *regexp.Regexp
	shouldRetry bool
}

// NewRetryIfContains creates a new retry condition based on output pattern.
func NewRetryIfContains(pattern string) (*RetryIfContains, error) {
	// Try to compile as regex, fall back to simple string matching
	var regex *regexp.Regexp
	regex, _ = regexp.Compile(pattern) // Ignore error, will use string matching if invalid
	
	return &RetryIfContains{
		pattern:    pattern,
		regex:      regex,
		shouldRetry: true, // Initially true so the first attempt runs
	}, nil
}

// GetCtx returns a background context.
func (r *RetryIfContains) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if we should stop retrying.
func (r *RetryIfContains) IsLimitReached() bool {
	return !r.shouldRetry
}

// StartTry does nothing for retry pattern condition.
func (r *RetryIfContains) StartTry() {}

// EndTry does nothing for retry pattern condition.
func (r *RetryIfContains) EndTry() {}

// SetLastExitCode is not used by retry pattern condition.
func (r *RetryIfContains) SetLastExitCode(_ int) {}

// SetLastOutput updates the output and checks if we should retry.
func (r *RetryIfContains) SetLastOutput(stdout, stderr string) {
	// Combine stdout and stderr for pattern matching
	combined := stdout + stderr
	
	// Check if pattern matches
	var matches bool
	if r.regex != nil {
		matches = r.regex.MatchString(combined)
	} else {
		// Fall back to simple string contains
		matches = strings.Contains(combined, r.pattern)
	}
	
	// Retry if pattern IS found
	r.shouldRetry = matches
}

// RetryRegex implements retry logic based on regex pattern matching.
//
//nolint:revive // Prefix is meaningful to distinguish from stop conditions
type RetryRegex struct {
	pattern    string
	regex      *regexp.Regexp
	shouldRetry bool
}

// NewRetryRegex creates a new retry condition based on regex pattern.
func NewRetryRegex(pattern string) (*RetryRegex, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	return &RetryRegex{
		pattern:    pattern,
		regex:      regex,
		shouldRetry: true, // Initially true so the first attempt runs
	}, nil
}

// GetCtx returns a background context.
func (r *RetryRegex) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if we should stop retrying.
func (r *RetryRegex) IsLimitReached() bool {
	return !r.shouldRetry
}

// StartTry does nothing for retry regex condition.
func (r *RetryRegex) StartTry() {}

// EndTry does nothing for retry regex condition.
func (r *RetryRegex) EndTry() {}

// SetLastExitCode is not used by retry regex condition.
func (r *RetryRegex) SetLastExitCode(_ int) {}

// SetLastOutput updates the output and checks if we should retry.
func (r *RetryRegex) SetLastOutput(stdout, stderr string) {
	// Combine stdout and stderr for pattern matching
	combined := stdout + stderr
	
	// Retry if regex matches
	r.shouldRetry = r.regex.MatchString(combined)
}