package retry

import (
	"context"
	"testing"
)

func TestNewStopOnExitCode(t *testing.T) {
	codes := []int{0, 1, 2}
	condition := NewStopOnExitCode(codes)

	if condition == nil {
		t.Fatal("NewStopOnExitCode should return non-nil condition")
	}

	if len(condition.stopCodes) != 3 {
		t.Errorf("Expected 3 stop codes, got %d", len(condition.stopCodes))
	}

	if condition.lastExitCode != -1 {
		t.Errorf("Expected initial lastExitCode to be -1, got %d", condition.lastExitCode)
	}

	if condition.shouldStop {
		t.Error("Expected initial shouldStop to be false")
	}
}

func TestStopOnExitCode_GetCtx(t *testing.T) {
	condition := NewStopOnExitCode([]int{1})
	ctx := condition.GetCtx()

	if ctx != context.Background() {
		t.Error("GetCtx() should return background context")
	}
}

func TestStopOnExitCode_IsLimitReached_Initial(t *testing.T) {
	condition := NewStopOnExitCode([]int{1})

	if condition.IsLimitReached() {
		t.Error("IsLimitReached() should return false initially")
	}
}

func TestStopOnExitCode_SetLastExitCode_Match(t *testing.T) {
	condition := NewStopOnExitCode([]int{1, 2, 3})

	// Test matching exit code
	condition.SetLastExitCode(2)

	if !condition.IsLimitReached() {
		t.Error("IsLimitReached() should return true after matching exit code")
	}

	if condition.lastExitCode != 2 {
		t.Errorf("Expected lastExitCode to be 2, got %d", condition.lastExitCode)
	}
}

func TestStopOnExitCode_SetLastExitCode_NoMatch(t *testing.T) {
	condition := NewStopOnExitCode([]int{1, 2, 3})

	// Test non-matching exit code
	condition.SetLastExitCode(5)

	if condition.IsLimitReached() {
		t.Error("IsLimitReached() should return false for non-matching exit code")
	}

	if condition.lastExitCode != 5 {
		t.Errorf("Expected lastExitCode to be 5, got %d", condition.lastExitCode)
	}
}

func TestStopOnExitCode_SetLastExitCode_Multiple(t *testing.T) {
	condition := NewStopOnExitCode([]int{1, 2, 3})

	// Test sequence of exit codes
	condition.SetLastExitCode(5) // No match
	if condition.IsLimitReached() {
		t.Error("Should not stop on non-matching code")
	}

	condition.SetLastExitCode(1) // Match
	if !condition.IsLimitReached() {
		t.Error("Should stop on matching code")
	}

	// Reset state for next test
	condition.SetLastExitCode(4) // No match - should reset shouldStop
	if condition.IsLimitReached() {
		t.Error("Should not stop after non-matching code following a match")
	}
}

func TestStopOnExitCode_StartTryEndTry(t *testing.T) {
	condition := NewStopOnExitCode([]int{1})

	// These methods should not panic and should be no-ops
	condition.StartTry()
	condition.EndTry()
}

func TestStopOnExitCode_SetLastOutput(t *testing.T) {
	condition := NewStopOnExitCode([]int{1})

	// This method should not panic and should be no-op
	condition.SetLastOutput("stdout", "stderr")
}

func TestStopOnExitCode_EdgeCases(t *testing.T) {
	// Test with empty stop codes
	condition := NewStopOnExitCode([]int{})
	condition.SetLastExitCode(1)
	if condition.IsLimitReached() {
		t.Error("Should not stop when no stop codes are configured")
	}

	// Test with negative exit codes
	condition = NewStopOnExitCode([]int{-1})
	condition.SetLastExitCode(-1)
	if !condition.IsLimitReached() {
		t.Error("Should stop on negative exit code match")
	}

	// Test with zero
	condition = NewStopOnExitCode([]int{0})
	condition.SetLastExitCode(0)
	if !condition.IsLimitReached() {
		t.Error("Should stop on zero exit code match")
	}
}