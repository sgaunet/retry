package retry

import (
	"context"
	"time"
)

type StopOnMaxExecutionTime struct {
	maxExecutionTime time.Duration
	tries            uint
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewStopOnMaxExecTime(maxExecTime time.Duration) *StopOnMaxExecutionTime {
	s := &StopOnMaxExecutionTime{maxExecutionTime: maxExecTime}
	s.ctx, s.cancel = context.WithTimeout(context.Background(), maxExecTime)
	return s
}

func (s *StopOnMaxExecutionTime) GetCtx() context.Context {
	return s.ctx
}

func (s *StopOnMaxExecutionTime) IsLimitReached() bool {
	return s.ctx.Err() != nil
}

func (s *StopOnMaxExecutionTime) StartTry() {
	s.tries++
}

func (s *StopOnMaxExecutionTime) EndTry() {
	s.cancel()
}
