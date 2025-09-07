package retry

import (
	"context"
	"testing"
	"time"
)

func TestNewStopAtTimeOfDay_Valid(t *testing.T) {
	timeStr := "14:30"
	condition, err := NewStopAtTimeOfDay(timeStr)

	if err != nil {
		t.Fatalf("NewStopAtTimeOfDay should not return error for valid time: %v", err)
	}

	if condition == nil {
		t.Fatal("NewStopAtTimeOfDay should return non-nil condition")
	}

	// Check that stop time has correct hour and minute
	if condition.stopTime.Hour() != 14 {
		t.Errorf("Expected hour 14, got %d", condition.stopTime.Hour())
	}

	if condition.stopTime.Minute() != 30 {
		t.Errorf("Expected minute 30, got %d", condition.stopTime.Minute())
	}
}

func TestNewStopAtTimeOfDay_Invalid(t *testing.T) {
	testCases := []string{
		"25:00",    // Invalid hour
		"12:60",    // Invalid minute  
		"12:30:45", // Too many parts
		"12",       // Missing minute
		"abc",      // Not a time
		"",         // Empty string
	}

	for _, timeStr := range testCases {
		condition, err := NewStopAtTimeOfDay(timeStr)

		if err == nil {
			t.Errorf("NewStopAtTimeOfDay should return error for invalid time %q", timeStr)
		}

		if condition != nil {
			t.Errorf("NewStopAtTimeOfDay should return nil condition for invalid time %q", timeStr)
		}
	}
}

func TestNewStopAtTimeOfDay_FutureTime(t *testing.T) {
	now := time.Now()
	// Create a time 2 hours from now
	futureTime := now.Add(2 * time.Hour)
	timeStr := futureTime.Format("15:04")

	condition, err := NewStopAtTimeOfDay(timeStr)
	if err != nil {
		t.Fatalf("NewStopAtTimeOfDay should not return error: %v", err)
	}

	// Stop time should be today
	if condition.stopTime.Day() != now.Day() {
		t.Errorf("Expected stop time to be today, but got day %d", condition.stopTime.Day())
	}

	// Should not have reached the limit yet
	if condition.IsLimitReached() {
		t.Error("Should not have reached limit for future time")
	}
}

func TestNewStopAtTimeOfDay_PastTime(t *testing.T) {
	now := time.Now()
	// Create a time 2 hours ago
	pastTime := now.Add(-2 * time.Hour)
	timeStr := pastTime.Format("15:04")

	condition, err := NewStopAtTimeOfDay(timeStr)
	if err != nil {
		t.Fatalf("NewStopAtTimeOfDay should not return error: %v", err)
	}

	// Stop time should be tomorrow since the time has passed today
	expectedDay := now.Add(24 * time.Hour).Day()
	if condition.stopTime.Day() != expectedDay {
		t.Errorf("Expected stop time to be tomorrow (day %d), but got day %d", expectedDay, condition.stopTime.Day())
	}

	// Should not have reached the limit yet
	if condition.IsLimitReached() {
		t.Error("Should not have reached limit for time set to tomorrow")
	}
}

func TestStopAtTimeOfDay_GetCtx(t *testing.T) {
	condition, _ := NewStopAtTimeOfDay("23:59")
	ctx := condition.GetCtx()

	if ctx != context.Background() {
		t.Error("GetCtx() should return background context")
	}
}

func TestStopAtTimeOfDay_IsLimitReached_NotReached(t *testing.T) {
	// Set time far in the future
	condition, _ := NewStopAtTimeOfDay("23:59")

	if condition.IsLimitReached() {
		t.Error("IsLimitReached() should return false for future time")
	}
}

func TestStopAtTimeOfDay_IsLimitReached_ManualTest(t *testing.T) {
	// This test manually sets the stop time to the past to verify the logic
	condition, _ := NewStopAtTimeOfDay("23:59")
	
	// Manually set stop time to past for testing
	condition.stopTime = time.Now().Add(-1 * time.Minute)

	if !condition.IsLimitReached() {
		t.Error("IsLimitReached() should return true when stop time has passed")
	}
}

func TestStopAtTimeOfDay_StartTryEndTry(t *testing.T) {
	condition, _ := NewStopAtTimeOfDay("23:59")

	// These methods should not panic and should be no-ops
	condition.StartTry()
	condition.EndTry()
}

func TestStopAtTimeOfDay_EdgeCases(t *testing.T) {
	// Test midnight
	condition, err := NewStopAtTimeOfDay("00:00")
	if err != nil {
		t.Errorf("Should handle midnight time: %v", err)
	}

	// Test noon
	condition, err = NewStopAtTimeOfDay("12:00")
	if err != nil {
		t.Errorf("Should handle noon time: %v", err)
	}

	// Test with leading zeros
	condition, err = NewStopAtTimeOfDay("09:05")
	if err != nil {
		t.Errorf("Should handle leading zeros: %v", err)
	}

	if condition.stopTime.Hour() != 9 || condition.stopTime.Minute() != 5 {
		t.Errorf("Expected 09:05, got %02d:%02d", condition.stopTime.Hour(), condition.stopTime.Minute())
	}
}

func TestStopAtTimeOfDay_TimeZone(t *testing.T) {
	condition, _ := NewStopAtTimeOfDay("12:00")

	// Stop time should be in the same timezone as current time
	now := time.Now()
	if condition.stopTime.Location() != now.Location() {
		t.Error("Stop time should be in the same timezone as current time")
	}
}