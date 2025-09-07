package retry

import (
	"context"
	"testing"
	"time"
)

func TestStopOnTimeout_GetCtx(t *testing.T) {
	timeout := 5 * time.Second
	condition := NewStopOnTimeout(timeout)

	ctx := condition.GetCtx()
	if ctx == nil {
		t.Fatal("GetCtx() should return non-nil context")
	}

	// Check if context has timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("Context should have deadline")
	}

	// Deadline should be approximately now + timeout (with some tolerance)
	expectedDeadline := time.Now().Add(timeout)
	tolerance := 100 * time.Millisecond
	if deadline.Before(expectedDeadline.Add(-tolerance)) || deadline.After(expectedDeadline.Add(tolerance)) {
		t.Errorf("Deadline %v not within tolerance of expected %v", deadline, expectedDeadline)
	}
}

func TestStopOnTimeout_IsLimitReached_NotReached(t *testing.T) {
	timeout := 1 * time.Second
	condition := NewStopOnTimeout(timeout)

	if condition.IsLimitReached() {
		t.Error("IsLimitReached() should return false immediately after creation")
	}
}

func TestStopOnTimeout_IsLimitReached_Reached(t *testing.T) {
	timeout := 10 * time.Millisecond
	condition := NewStopOnTimeout(timeout)

	// Wait for timeout to elapse
	time.Sleep(20 * time.Millisecond)

	if !condition.IsLimitReached() {
		t.Error("IsLimitReached() should return true after timeout elapsed")
	}
}

func TestStopOnTimeout_ContextCancellation(t *testing.T) {
	timeout := 50 * time.Millisecond
	condition := NewStopOnTimeout(timeout)

	ctx := condition.GetCtx()

	// Initially context should not be cancelled
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled immediately")
	default:
		// Expected
	}

	// Wait for context to be cancelled due to timeout
	select {
	case <-ctx.Done():
		// Expected - context should be cancelled after timeout
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should have been cancelled after timeout")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected context error to be DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestStopOnTimeout_StartTryEndTry(t *testing.T) {
	condition := NewStopOnTimeout(1 * time.Second)

	// These methods should not panic and should be no-ops
	condition.StartTry()
	condition.EndTry()
}