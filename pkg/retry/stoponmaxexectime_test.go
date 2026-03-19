package retry

import (
	"testing"
	"time"
)

func TestStopOnMaxExecutionTime_EndTryIsNoop(t *testing.T) {
	// Bug 2: EndTry() was cancelling context prematurely.
	// Verify that multiple StartTry/EndTry cycles do not cancel the context.
	s := NewStopOnMaxExecTime(5 * time.Second)
	defer s.Cancel()

	for i := 0; i < 5; i++ {
		s.StartTry()
		s.EndTry()
	}

	if s.GetCtx().Err() != nil {
		t.Errorf("context should not be cancelled after EndTry calls, got: %v", s.GetCtx().Err())
	}
}

func TestStopOnMaxExecutionTime_IsLimitReachedFalseBeforeTimeout(t *testing.T) {
	// Bug 2: IsLimitReached() must return false before the timeout expires,
	// even after multiple StartTry/EndTry cycles.
	s := NewStopOnMaxExecTime(5 * time.Second)
	defer s.Cancel()

	for i := 0; i < 3; i++ {
		s.StartTry()
		s.EndTry()
	}

	if s.IsLimitReached() {
		t.Error("IsLimitReached() should return false before timeout expires")
	}
}

func TestStopOnMaxExecutionTime_CancelCancelsContext(t *testing.T) {
	// Bug 2: Cancel() must cancel the timeout context for explicit cleanup.
	s := NewStopOnMaxExecTime(5 * time.Second)

	s.StartTry()
	s.EndTry()

	// Context should still be alive before Cancel.
	if s.GetCtx().Err() != nil {
		t.Fatalf("context should be alive before Cancel(), got: %v", s.GetCtx().Err())
	}

	s.Cancel()

	if s.GetCtx().Err() == nil {
		t.Error("context should be cancelled after calling Cancel()")
	}
}
