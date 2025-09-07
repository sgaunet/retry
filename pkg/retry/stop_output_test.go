package retry

import (
	"context"
	"testing"
)

func TestNewStopOnOutputContains(t *testing.T) {
	pattern := "success"
	condition, err := NewStopOnOutputContains(pattern)

	if err != nil {
		t.Fatalf("NewStopOnOutputContains should not return error: %v", err)
	}

	if condition == nil {
		t.Fatal("NewStopOnOutputContains should return non-nil condition")
	}

	if condition.pattern != pattern {
		t.Errorf("Expected pattern %q, got %q", pattern, condition.pattern)
	}

	if !condition.contains {
		t.Error("Expected contains to be true")
	}

	if condition.shouldStop {
		t.Error("Expected initial shouldStop to be false")
	}
}

func TestNewStopOnOutputNotContains(t *testing.T) {
	pattern := "error"
	condition, err := NewStopOnOutputNotContains(pattern)

	if err != nil {
		t.Fatalf("NewStopOnOutputNotContains should not return error: %v", err)
	}

	if condition == nil {
		t.Fatal("NewStopOnOutputNotContains should return non-nil condition")
	}

	if condition.pattern != pattern {
		t.Errorf("Expected pattern %q, got %q", pattern, condition.pattern)
	}

	if condition.contains {
		t.Error("Expected contains to be false")
	}

	if condition.shouldStop {
		t.Error("Expected initial shouldStop to be false")
	}
}

func TestStopOnOutputPattern_GetCtx(t *testing.T) {
	condition, _ := NewStopOnOutputContains("test")
	ctx := condition.GetCtx()

	if ctx != context.Background() {
		t.Error("GetCtx() should return background context")
	}
}

func TestStopOnOutputPattern_IsLimitReached_Initial(t *testing.T) {
	condition, _ := NewStopOnOutputContains("test")

	if condition.IsLimitReached() {
		t.Error("IsLimitReached() should return false initially")
	}
}

func TestStopOnOutputPattern_SetLastOutput_Contains_Match(t *testing.T) {
	condition, _ := NewStopOnOutputContains("success")

	// Test matching output in stdout
	condition.SetLastOutput("Operation was successful", "")
	if !condition.IsLimitReached() {
		t.Error("Should stop when output contains the pattern")
	}

	// Reset for next test
	condition.shouldStop = false
	
	// Test matching output in stderr
	condition.SetLastOutput("", "success message")
	if !condition.IsLimitReached() {
		t.Error("Should stop when stderr contains the pattern")
	}
}

func TestStopOnOutputPattern_SetLastOutput_Contains_NoMatch(t *testing.T) {
	condition, _ := NewStopOnOutputContains("success")

	// Test non-matching output
	condition.SetLastOutput("Operation failed", "error occurred")
	if condition.IsLimitReached() {
		t.Error("Should not stop when output doesn't contain the pattern")
	}
}

func TestStopOnOutputPattern_SetLastOutput_NotContains_Match(t *testing.T) {
	condition, _ := NewStopOnOutputNotContains("error")

	// Test output without error (should stop)
	condition.SetLastOutput("Operation successful", "all good")
	if !condition.IsLimitReached() {
		t.Error("Should stop when output doesn't contain the pattern")
	}
}

func TestStopOnOutputPattern_SetLastOutput_NotContains_NoMatch(t *testing.T) {
	condition, _ := NewStopOnOutputNotContains("error")

	// Test output with error (should not stop)
	condition.SetLastOutput("Operation failed", "error occurred")
	if condition.IsLimitReached() {
		t.Error("Should not stop when output contains the pattern we want to avoid")
	}
}

func TestStopOnOutputPattern_RegexPattern(t *testing.T) {
	// Test with regex pattern
	condition, _ := NewStopOnOutputContains(`\d+ files processed`)

	// Test matching regex
	condition.SetLastOutput("Successfully processed 42 files processed", "")
	if !condition.IsLimitReached() {
		t.Error("Should stop when output matches regex pattern")
	}

	// Reset for next test - create new condition since shouldStop is not public
	condition, _ = NewStopOnOutputContains(`\d+ files processed`)

	// Test non-matching regex
	condition.SetLastOutput("No files were processed", "")
	if condition.IsLimitReached() {
		t.Error("Should not stop when output doesn't match regex pattern")
	}
}

func TestStopOnOutputPattern_InvalidRegex(t *testing.T) {
	// Test with invalid regex - should fall back to string matching
	condition, _ := NewStopOnOutputContains("[invalid")

	// Should use string matching instead of regex
	condition.SetLastOutput("Found [invalid pattern", "")
	if !condition.IsLimitReached() {
		t.Error("Should stop using string matching when regex is invalid")
	}
}

func TestStopOnOutputPattern_CombinedOutput(t *testing.T) {
	condition, _ := NewStopOnOutputContains("success")

	// Test pattern in combined stdout+stderr
	condition.SetLastOutput("Operation ", "completed with success")
	if !condition.IsLimitReached() {
		t.Error("Should stop when pattern is found in combined output")
	}
}

func TestStopOnOutputPattern_StartTryEndTry(t *testing.T) {
	condition, _ := NewStopOnOutputContains("test")

	// These methods should not panic and should be no-ops
	condition.StartTry()
	condition.EndTry()
}

func TestStopOnOutputPattern_SetLastExitCode(t *testing.T) {
	condition, _ := NewStopOnOutputContains("test")

	// This method should not panic and should be no-op
	condition.SetLastExitCode(0)
}

func TestStopOnOutputPattern_EmptyPattern(t *testing.T) {
	condition, _ := NewStopOnOutputContains("")

	// Empty pattern should match everything
	condition.SetLastOutput("any output", "any error")
	if !condition.IsLimitReached() {
		t.Error("Empty pattern should match any output")
	}
}

func TestStopOnOutputPattern_EmptyOutput(t *testing.T) {
	condition, _ := NewStopOnOutputNotContains("error")

	// Empty output should not contain error, so should stop
	condition.SetLastOutput("", "")
	if !condition.IsLimitReached() {
		t.Error("Empty output should not contain any pattern")
	}
}