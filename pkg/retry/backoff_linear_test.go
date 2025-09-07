package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLinearBackoff_NextDelay(t *testing.T) {
	tests := []struct {
		name      string
		baseDelay time.Duration
		increment time.Duration
		maxDelay  time.Duration
		attempt   int
		expected  time.Duration
	}{
		{
			name:      "first attempt returns base delay",
			baseDelay: 1 * time.Second,
			increment: 500 * time.Millisecond,
			maxDelay:  10 * time.Second,
			attempt:   1,
			expected:  1 * time.Second,
		},
		{
			name:      "second attempt adds one increment",
			baseDelay: 1 * time.Second,
			increment: 500 * time.Millisecond,
			maxDelay:  10 * time.Second,
			attempt:   2,
			expected:  1500 * time.Millisecond,
		},
		{
			name:      "third attempt adds two increments",
			baseDelay: 1 * time.Second,
			increment: 500 * time.Millisecond,
			maxDelay:  10 * time.Second,
			attempt:   3,
			expected:  2 * time.Second,
		},
		{
			name:      "respects max delay cap",
			baseDelay: 1 * time.Second,
			increment: 2 * time.Second,
			maxDelay:  3 * time.Second,
			attempt:   5,
			expected:  3 * time.Second,
		},
		{
			name:      "zero attempt returns base delay",
			baseDelay: 1 * time.Second,
			increment: 500 * time.Millisecond,
			maxDelay:  10 * time.Second,
			attempt:   0,
			expected:  1 * time.Second,
		},
		{
			name:      "negative attempt returns base delay",
			baseDelay: 1 * time.Second,
			increment: 500 * time.Millisecond,
			maxDelay:  10 * time.Second,
			attempt:   -1,
			expected:  1 * time.Second,
		},
		{
			name:      "no max delay allows unbounded growth",
			baseDelay: 1 * time.Second,
			increment: 1 * time.Second,
			maxDelay:  0,
			attempt:   10,
			expected:  10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := NewLinearBackoff(tt.baseDelay, tt.increment, tt.maxDelay)
			actual := lb.NextDelay(tt.attempt)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestLinearBackoff_Sequence(t *testing.T) {
	lb := NewLinearBackoff(100*time.Millisecond, 50*time.Millisecond, 500*time.Millisecond)
	
	expectedSequence := []time.Duration{
		100 * time.Millisecond, // attempt 1: base
		150 * time.Millisecond, // attempt 2: base + 1*increment
		200 * time.Millisecond, // attempt 3: base + 2*increment
		250 * time.Millisecond, // attempt 4: base + 3*increment
		300 * time.Millisecond, // attempt 5: base + 4*increment
		350 * time.Millisecond, // attempt 6: base + 5*increment
		400 * time.Millisecond, // attempt 7: base + 6*increment
		450 * time.Millisecond, // attempt 8: base + 7*increment
		500 * time.Millisecond, // attempt 9: capped at max
		500 * time.Millisecond, // attempt 10: capped at max
	}
	
	for i, expected := range expectedSequence {
		actual := lb.NextDelay(i + 1)
		assert.Equal(t, expected, actual, "Attempt %d", i+1)
	}
}