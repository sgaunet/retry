package retry

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// SuccessOnExitCode implements success logic based on specific exit codes.
// It considers specific exit codes as success, not just 0.
type SuccessOnExitCode struct {
	successCodes []int
	lastExitCode int
	isSuccess    bool
}

// NewSuccessOnExitCode creates a new success condition based on exit codes.
func NewSuccessOnExitCode(codes []int) *SuccessOnExitCode {
	return &SuccessOnExitCode{
		successCodes: codes,
		lastExitCode: -1,
		isSuccess:    false,
	}
}

// GetCtx returns a background context.
func (s *SuccessOnExitCode) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if success has been achieved.
func (s *SuccessOnExitCode) IsLimitReached() bool {
	return s.isSuccess
}

// StartTry does nothing for success exit code condition.
func (s *SuccessOnExitCode) StartTry() {}

// EndTry does nothing for success exit code condition.
func (s *SuccessOnExitCode) EndTry() {}

// SetLastExitCode updates the last exit code and checks for success.
func (s *SuccessOnExitCode) SetLastExitCode(code int) {
	s.lastExitCode = code
	s.isSuccess = false
	
	// Check if this exit code is in our success list
	for _, successCode := range s.successCodes {
		if code == successCode {
			s.isSuccess = true
			return
		}
	}
}

// SetLastOutput is not used by success exit code condition.
func (s *SuccessOnExitCode) SetLastOutput(_, _ string) {}

// SuccessContains implements success logic based on output pattern matching.
// It considers the command successful if output contains a specific pattern.
type SuccessContains struct {
	pattern   string
	regex     *regexp.Regexp
	isSuccess bool
}

// NewSuccessContains creates a new success condition based on output pattern.
func NewSuccessContains(pattern string) (*SuccessContains, error) {
	// Try to compile as regex, fall back to simple string matching
	var regex *regexp.Regexp
	regex, _ = regexp.Compile(pattern) // Ignore error, will use string matching if invalid
	
	return &SuccessContains{
		pattern:   pattern,
		regex:     regex,
		isSuccess: false,
	}, nil
}

// GetCtx returns a background context.
func (s *SuccessContains) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if success has been achieved.
func (s *SuccessContains) IsLimitReached() bool {
	return s.isSuccess
}

// StartTry does nothing for success pattern condition.
func (s *SuccessContains) StartTry() {}

// EndTry does nothing for success pattern condition.
func (s *SuccessContains) EndTry() {}

// SetLastExitCode is not used by success pattern condition.
func (s *SuccessContains) SetLastExitCode(_ int) {}

// SetLastOutput updates the output and checks for success.
func (s *SuccessContains) SetLastOutput(stdout, stderr string) {
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
	
	// Success if pattern IS found
	s.isSuccess = matches
}

// FailIfContains implements immediate failure logic based on output pattern.
// It causes immediate failure if output contains a specific pattern.
type FailIfContains struct {
	pattern    string
	regex      *regexp.Regexp
	shouldFail bool
}

// NewFailIfContains creates a new fail condition based on output pattern.
func NewFailIfContains(pattern string) (*FailIfContains, error) {
	// Try to compile as regex, fall back to simple string matching
	var regex *regexp.Regexp
	regex, _ = regexp.Compile(pattern) // Ignore error, will use string matching if invalid
	
	return &FailIfContains{
		pattern:    pattern,
		regex:      regex,
		shouldFail: false,
	}, nil
}

// GetCtx returns a background context.
func (f *FailIfContains) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if we should fail immediately.
func (f *FailIfContains) IsLimitReached() bool {
	return f.shouldFail
}

// StartTry does nothing for fail pattern condition.
func (f *FailIfContains) StartTry() {}

// EndTry does nothing for fail pattern condition.
func (f *FailIfContains) EndTry() {}

// SetLastExitCode is not used by fail pattern condition.
func (f *FailIfContains) SetLastExitCode(_ int) {}

// SetLastOutput updates the output and checks if we should fail.
func (f *FailIfContains) SetLastOutput(stdout, stderr string) {
	// Combine stdout and stderr for pattern matching
	combined := stdout + stderr
	
	// Check if pattern matches
	var matches bool
	if f.regex != nil {
		matches = f.regex.MatchString(combined)
	} else {
		// Fall back to simple string contains
		matches = strings.Contains(combined, f.pattern)
	}
	
	// Fail immediately if pattern IS found
	f.shouldFail = matches
}

// SuccessRegex implements success logic based on regex pattern matching.
type SuccessRegex struct {
	pattern   string
	regex     *regexp.Regexp
	isSuccess bool
}

// NewSuccessRegex creates a new success condition based on regex pattern.
func NewSuccessRegex(pattern string) (*SuccessRegex, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	return &SuccessRegex{
		pattern:   pattern,
		regex:     regex,
		isSuccess: false,
	}, nil
}

// GetCtx returns a background context.
func (s *SuccessRegex) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if success has been achieved.
func (s *SuccessRegex) IsLimitReached() bool {
	return s.isSuccess
}

// StartTry does nothing for success regex condition.
func (s *SuccessRegex) StartTry() {}

// EndTry does nothing for success regex condition.
func (s *SuccessRegex) EndTry() {}

// SetLastExitCode is not used by success regex condition.
func (s *SuccessRegex) SetLastExitCode(_ int) {}

// SetLastOutput updates the output and checks for success.
func (s *SuccessRegex) SetLastOutput(stdout, stderr string) {
	// Combine stdout and stderr for pattern matching
	combined := stdout + stderr
	
	// Success if regex matches
	s.isSuccess = s.regex.MatchString(combined)
}