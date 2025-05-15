package retry

import (
	"context"
	"time"
)

// StopOnMaxExecutionTime is a retry condition that stops when the maximum execution time is reached.
// It implements the ConditionRetryer interface.
// It uses a context with timeout to determine when to stop retrying.
type StopOnMaxExecutionTime struct {
	maxExecutionTime time.Duration
	tries            uint
	ctx              context.Context //nolint:containedctx
	cancel           context.CancelFunc
}

// NewStopOnMaxExecTime creates a new StopOnMaxExecutionTime instance with the given maximum execution time.
func NewStopOnMaxExecTime(maxExecTime time.Duration) *StopOnMaxExecutionTime {
	s := &StopOnMaxExecutionTime{maxExecutionTime: maxExecTime}
	s.ctx, s.cancel = context.WithTimeout(context.Background(), maxExecTime)
	return s
}

// GetCtx returns the context of the StopOnMaxExecutionTime instance.
func (s *StopOnMaxExecutionTime) GetCtx() context.Context {
	return s.ctx
}

// IsLimitReached checks if the maximum execution time has been reached.
func (s *StopOnMaxExecutionTime) IsLimitReached() bool {
	return s.ctx.Err() != nil
}

// StartTry increments the number of tries.
func (s *StopOnMaxExecutionTime) StartTry() {
	s.tries++
}

// EndTry cancels the context, stopping the retry process.
func (s *StopOnMaxExecutionTime) EndTry() {
	s.cancel()
}
