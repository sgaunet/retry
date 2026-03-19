package retry

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// Test_FailIfContains_ExitZero verifies Bug 3:
// When a command exits 0 but its output matches a fail pattern,
// RunWithLogger must return ErrFailConditionMet.
func Test_FailIfContains_ExitZero(t *testing.T) {
	defer goleak.VerifyNone(t)

	failCond, err := NewFailIfContains("FATAL")
	if err != nil {
		t.Fatalf("NewFailIfContains: %v", err)
	}

	// Use a composite so both stop (max-tries) and the fail condition are evaluated.
	stopCond := NewStopOnMaxTries(3)
	composite := NewCompositeCondition(LogicOR, stopCond, failCond)

	// Command exits 0 but prints the fail pattern on stdout.
	r, err := NewRetry("bash -c 'echo FATAL; exit 0'", composite)
	if err != nil {
		t.Fatalf("NewRetry: %v", err)
	}
	r.SetOutputWriters(io.Discard, io.Discard)

	retryErr := r.RunWithLogger(context.Background(), nil)
	if !errors.Is(retryErr, ErrFailConditionMet) {
		t.Errorf("expected ErrFailConditionMet, got: %v", retryErr)
	}
}

func Test_retry_SetBackoffStrategy(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := NewRetry("", NewStopOnMaxTries(3))
	if err != nil {
		t.Errorf("Expected no error")
	}
	strategy := NewFixedBackoff(5 * time.Second)
	r.SetBackoffStrategy(strategy)
	if r.backoff == nil {
		t.Errorf("Expected backoff strategy to be set")
	}
}
