package retry

import (
	"context"
	"slices"
)

// StopOnExitCode stops retrying when a specific exit code is encountered.
type StopOnExitCode struct {
	stopCodes    []int
	lastExitCode int
	shouldStop   bool
}

// NewStopOnExitCode creates a new exit code based stop condition.
func NewStopOnExitCode(codes []int) *StopOnExitCode {
	return &StopOnExitCode{
		stopCodes:    codes,
		lastExitCode: -1,
		shouldStop:   false,
	}
}

// GetCtx returns a background context as exit code checking doesn't need timeout.
func (s *StopOnExitCode) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if we should stop based on the last exit code.
func (s *StopOnExitCode) IsLimitReached() bool {
	return s.shouldStop
}

// StartTry does nothing for exit code condition.
func (s *StopOnExitCode) StartTry() {}

// EndTry does nothing for exit code condition.
func (s *StopOnExitCode) EndTry() {}

// SetLastExitCode updates the last exit code and checks if we should stop.
func (s *StopOnExitCode) SetLastExitCode(code int) {
	s.lastExitCode = code
	s.shouldStop = slices.Contains(s.stopCodes, code)
}

// SetLastOutput is not used by exit code condition.
func (s *StopOnExitCode) SetLastOutput(_, _ string) {
	// Not used by exit code condition
}