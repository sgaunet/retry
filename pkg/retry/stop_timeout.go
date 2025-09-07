package retry

import (
	"context"
	"time"
)

// StopOnTimeout is a simpler timeout-based stop condition.
// Unlike StopOnMaxExecutionTime, this is more straightforward for CLI usage.
type StopOnTimeout struct {
	timeout   time.Duration
	startTime time.Time
	ctx       context.Context //nolint:containedctx // Required for timeout management
	cancel    context.CancelFunc
}

// NewStopOnTimeout creates a new timeout-based stop condition.
func NewStopOnTimeout(timeout time.Duration) *StopOnTimeout {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &StopOnTimeout{
		timeout:   timeout,
		startTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// GetCtx returns the context with timeout.
func (s *StopOnTimeout) GetCtx() context.Context {
	return s.ctx
}

// IsLimitReached checks if the timeout has been exceeded.
func (s *StopOnTimeout) IsLimitReached() bool {
	return time.Since(s.startTime) >= s.timeout || s.ctx.Err() != nil
}

// StartTry does nothing for timeout condition.
func (s *StopOnTimeout) StartTry() {}

// EndTry does nothing for timeout condition.
func (s *StopOnTimeout) EndTry() {}

// Cancel cancels the timeout context.
func (s *StopOnTimeout) Cancel() {
	s.cancel()
}