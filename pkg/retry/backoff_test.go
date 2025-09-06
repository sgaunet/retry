package retry

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestFixedBackoff_NextDelay(t *testing.T) {
	defer goleak.VerifyNone(t)
	
	tests := []struct {
		name    string
		delay   time.Duration
		attempt int
		want    time.Duration
	}{
		{
			name:    "zero delay",
			delay:   0,
			attempt: 1,
			want:    0,
		},
		{
			name:    "1 second delay",
			delay:   time.Second,
			attempt: 1,
			want:    time.Second,
		},
		{
			name:    "same delay for multiple attempts",
			delay:   2 * time.Second,
			attempt: 5,
			want:    2 * time.Second,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFixedBackoff(tt.delay)
			if got := f.NextDelay(tt.attempt); got != tt.want {
				t.Errorf("FixedBackoff.NextDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExponentialBackoff_NextDelay(t *testing.T) {
	defer goleak.VerifyNone(t)
	
	tests := []struct {
		name       string
		baseDelay  time.Duration
		maxDelay   time.Duration
		multiplier float64
		attempt    int
		want       time.Duration
	}{
		{
			name:       "first attempt",
			baseDelay:  time.Second,
			maxDelay:   time.Minute,
			multiplier: 2.0,
			attempt:    1,
			want:       time.Second,
		},
		{
			name:       "second attempt",
			baseDelay:  time.Second,
			maxDelay:   time.Minute,
			multiplier: 2.0,
			attempt:    2,
			want:       2 * time.Second,
		},
		{
			name:       "third attempt",
			baseDelay:  time.Second,
			maxDelay:   time.Minute,
			multiplier: 2.0,
			attempt:    3,
			want:       4 * time.Second,
		},
		{
			name:       "zero or negative attempt",
			baseDelay:  time.Second,
			maxDelay:   time.Minute,
			multiplier: 2.0,
			attempt:    0,
			want:       time.Second,
		},
		{
			name:       "capped at max delay",
			baseDelay:  time.Second,
			maxDelay:   5 * time.Second,
			multiplier: 2.0,
			attempt:    10,
			want:       5 * time.Second,
		},
		{
			name:       "different multiplier",
			baseDelay:  100 * time.Millisecond,
			maxDelay:   time.Minute,
			multiplier: 1.5,
			attempt:    3,
			want:       time.Duration(float64(100*time.Millisecond) * 1.5 * 1.5),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExponentialBackoff(tt.baseDelay, tt.maxDelay, tt.multiplier)
			got := e.NextDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("ExponentialBackoff.NextDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExponentialBackoff_Progressive(t *testing.T) {
	defer goleak.VerifyNone(t)
	
	e := NewExponentialBackoff(100*time.Millisecond, 10*time.Second, 2.0)
	
	expected := []time.Duration{
		100 * time.Millisecond, // attempt 1
		200 * time.Millisecond, // attempt 2
		400 * time.Millisecond, // attempt 3
		800 * time.Millisecond, // attempt 4
		1600 * time.Millisecond, // attempt 5
	}
	
	for i, want := range expected {
		attempt := i + 1
		got := e.NextDelay(attempt)
		if got != want {
			t.Errorf("ExponentialBackoff.NextDelay(%d) = %v, want %v", attempt, got, want)
		}
	}
}

func TestExponentialBackoff_MaxDelayEnforcement(t *testing.T) {
	defer goleak.VerifyNone(t)
	
	e := NewExponentialBackoff(time.Second, 3*time.Second, 2.0)
	
	// Should grow: 1s, 2s, 4s, 8s, 16s...
	// But capped at 3s after the second attempt
	testCases := []struct {
		attempt int
		maxWant time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 3 * time.Second}, // Should be capped
		{4, 3 * time.Second}, // Should remain capped
		{10, 3 * time.Second}, // Should remain capped
	}
	
	for _, tc := range testCases {
		got := e.NextDelay(tc.attempt)
		if got > tc.maxWant {
			t.Errorf("ExponentialBackoff.NextDelay(%d) = %v, should not exceed %v", tc.attempt, got, tc.maxWant)
		}
		if tc.attempt >= 3 && got != tc.maxWant {
			t.Errorf("ExponentialBackoff.NextDelay(%d) = %v, want exactly %v (should be capped)", tc.attempt, got, tc.maxWant)
		}
	}
}

func TestNewFixedBackoff(t *testing.T) {
	defer goleak.VerifyNone(t)
	
	delay := 2 * time.Second
	f := NewFixedBackoff(delay)
	
	if f == nil {
		t.Error("NewFixedBackoff() returned nil")
	}
	
	if f.Delay != delay {
		t.Errorf("NewFixedBackoff().Delay = %v, want %v", f.Delay, delay)
	}
}

func TestNewExponentialBackoff(t *testing.T) {
	defer goleak.VerifyNone(t)
	
	baseDelay := time.Second
	maxDelay := time.Minute
	multiplier := 2.5
	
	e := NewExponentialBackoff(baseDelay, maxDelay, multiplier)
	
	if e == nil {
		t.Error("NewExponentialBackoff() returned nil")
	}
	
	if e.BaseDelay != baseDelay {
		t.Errorf("NewExponentialBackoff().BaseDelay = %v, want %v", e.BaseDelay, baseDelay)
	}
	
	if e.MaxDelay != maxDelay {
		t.Errorf("NewExponentialBackoff().MaxDelay = %v, want %v", e.MaxDelay, maxDelay)
	}
	
	if e.Multiplier != multiplier {
		t.Errorf("NewExponentialBackoff().Multiplier = %v, want %v", e.Multiplier, multiplier)
	}
}

func TestExponentialBackoff_EdgeCases(t *testing.T) {
	defer goleak.VerifyNone(t)
	
	t.Run("very large attempt number", func(t *testing.T) {
		e := NewExponentialBackoff(time.Millisecond, time.Hour, 2.0)
		got := e.NextDelay(100) // Very large attempt
		if got > time.Hour {
			t.Errorf("ExponentialBackoff.NextDelay(100) = %v, should not exceed MaxDelay %v", got, time.Hour)
		}
	})
	
	t.Run("multiplier close to 1", func(t *testing.T) {
		e := NewExponentialBackoff(time.Second, time.Minute, 1.1)
		delay1 := e.NextDelay(1)
		delay2 := e.NextDelay(2)
		
		if delay2 <= delay1 {
			t.Errorf("ExponentialBackoff should increase delays: %v should be > %v", delay2, delay1)
		}
	})
}