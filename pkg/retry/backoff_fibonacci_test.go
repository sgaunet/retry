package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFibonacciBackoff_NextDelay(t *testing.T) {
	tests := []struct {
		name      string
		baseDelay time.Duration
		maxDelay  time.Duration
		attempt   int
		expected  time.Duration
	}{
		{
			name:      "first attempt returns base delay (fib[0]=1)",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   1,
			expected:  1 * time.Second,
		},
		{
			name:      "second attempt returns base delay (fib[1]=1)",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   2,
			expected:  1 * time.Second,
		},
		{
			name:      "third attempt returns 2x base (fib[2]=2)",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   3,
			expected:  2 * time.Second,
		},
		{
			name:      "fourth attempt returns 3x base (fib[3]=3)",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   4,
			expected:  3 * time.Second,
		},
		{
			name:      "fifth attempt returns 5x base (fib[4]=5)",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   5,
			expected:  5 * time.Second,
		},
		{
			name:      "sixth attempt returns 8x base (fib[5]=8)",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   6,
			expected:  8 * time.Second,
		},
		{
			name:      "respects max delay cap",
			baseDelay: 1 * time.Second,
			maxDelay:  10 * time.Second,
			attempt:   10, // fib[9]=55, 55s > 10s max
			expected:  10 * time.Second,
		},
		{
			name:      "zero attempt returns base delay",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   0,
			expected:  1 * time.Second,
		},
		{
			name:      "negative attempt returns base delay",
			baseDelay: 1 * time.Second,
			maxDelay:  100 * time.Second,
			attempt:   -1,
			expected:  1 * time.Second,
		},
		{
			name:      "no max delay allows unbounded growth",
			baseDelay: 100 * time.Millisecond,
			maxDelay:  0,
			attempt:   8, // fib[7]=21
			expected:  2100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := NewFibonacciBackoff(tt.baseDelay, tt.maxDelay)
			actual := fb.NextDelay(tt.attempt)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestFibonacciBackoff_Sequence(t *testing.T) {
	fb := NewFibonacciBackoff(100*time.Millisecond, 2*time.Second)
	
	// Expected Fibonacci sequence: 1, 1, 2, 3, 5, 8, 13, 21, 34, 55...
	expectedSequence := []time.Duration{
		100 * time.Millisecond,  // attempt 1: 1 * base
		100 * time.Millisecond,  // attempt 2: 1 * base
		200 * time.Millisecond,  // attempt 3: 2 * base
		300 * time.Millisecond,  // attempt 4: 3 * base
		500 * time.Millisecond,  // attempt 5: 5 * base
		800 * time.Millisecond,  // attempt 6: 8 * base
		1300 * time.Millisecond, // attempt 7: 13 * base
		2000 * time.Millisecond, // attempt 8: 21 * base = 2.1s, capped at 2s
		2000 * time.Millisecond, // attempt 9: 34 * base = 3.4s, capped at 2s
		2000 * time.Millisecond, // attempt 10: 55 * base = 5.5s, capped at 2s
	}
	
	for i, expected := range expectedSequence {
		actual := fb.NextDelay(i + 1)
		assert.Equal(t, expected, actual, "Attempt %d", i+1)
	}
}

func TestFibonacciBackoff_SequenceConsistency(t *testing.T) {
	// Test that calling NextDelay multiple times with the same attempt
	// returns consistent results (i.e., the internal sequence is properly maintained)
	fb := NewFibonacciBackoff(1*time.Second, 100*time.Second)
	
	// Build up the sequence
	for i := 1; i <= 10; i++ {
		_ = fb.NextDelay(i)
	}
	
	// Now verify consistency
	assert.Equal(t, 1*time.Second, fb.NextDelay(1))
	assert.Equal(t, 1*time.Second, fb.NextDelay(2))
	assert.Equal(t, 2*time.Second, fb.NextDelay(3))
	assert.Equal(t, 5*time.Second, fb.NextDelay(5))
	assert.Equal(t, 55*time.Second, fb.NextDelay(10))
}