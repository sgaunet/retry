package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCustomBackoff_NextDelay(t *testing.T) {
	tests := []struct {
		name     string
		delays   []time.Duration
		attempt  int
		expected time.Duration
	}{
		{
			name: "first attempt returns first delay",
			delays: []time.Duration{
				1 * time.Second,
				2 * time.Second,
				5 * time.Second,
			},
			attempt:  1,
			expected: 1 * time.Second,
		},
		{
			name: "second attempt returns second delay",
			delays: []time.Duration{
				1 * time.Second,
				2 * time.Second,
				5 * time.Second,
			},
			attempt:  2,
			expected: 2 * time.Second,
		},
		{
			name: "third attempt returns third delay",
			delays: []time.Duration{
				1 * time.Second,
				2 * time.Second,
				5 * time.Second,
			},
			attempt:  3,
			expected: 5 * time.Second,
		},
		{
			name: "exceeding delays list returns last delay",
			delays: []time.Duration{
				1 * time.Second,
				2 * time.Second,
				5 * time.Second,
			},
			attempt:  5,
			expected: 5 * time.Second,
		},
		{
			name: "far exceeding delays list still returns last delay",
			delays: []time.Duration{
				1 * time.Second,
				2 * time.Second,
				5 * time.Second,
			},
			attempt:  100,
			expected: 5 * time.Second,
		},
		{
			name:     "empty delays returns 0",
			delays:   []time.Duration{},
			attempt:  1,
			expected: 0,
		},
		{
			name: "zero attempt returns 0",
			delays: []time.Duration{
				1 * time.Second,
				2 * time.Second,
			},
			attempt:  0,
			expected: 0,
		},
		{
			name: "negative attempt returns 0",
			delays: []time.Duration{
				1 * time.Second,
				2 * time.Second,
			},
			attempt:  -1,
			expected: 0,
		},
		{
			name: "single delay repeats",
			delays: []time.Duration{
				3 * time.Second,
			},
			attempt:  5,
			expected: 3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewCustomBackoff(tt.delays)
			actual := cb.NextDelay(tt.attempt)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCustomBackoff_ComplexSequence(t *testing.T) {
	// Test a typical escalating delay pattern
	delays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
	}
	
	cb := NewCustomBackoff(delays)
	
	// Test the full sequence
	for i, expected := range delays {
		actual := cb.NextDelay(i + 1)
		assert.Equal(t, expected, actual, "Attempt %d", i+1)
	}
	
	// Test beyond the sequence - should repeat last value
	assert.Equal(t, 30*time.Second, cb.NextDelay(9))
	assert.Equal(t, 30*time.Second, cb.NextDelay(10))
	assert.Equal(t, 30*time.Second, cb.NextDelay(20))
}

func TestCustomBackoff_VariousPatterns(t *testing.T) {
	t.Run("constant delays", func(t *testing.T) {
		// All delays are the same
		cb := NewCustomBackoff([]time.Duration{
			2 * time.Second,
			2 * time.Second,
			2 * time.Second,
		})
		assert.Equal(t, 2*time.Second, cb.NextDelay(1))
		assert.Equal(t, 2*time.Second, cb.NextDelay(2))
		assert.Equal(t, 2*time.Second, cb.NextDelay(3))
		assert.Equal(t, 2*time.Second, cb.NextDelay(10))
	})
	
	t.Run("decreasing delays", func(t *testing.T) {
		// Delays that decrease over time (unusual but valid)
		cb := NewCustomBackoff([]time.Duration{
			10 * time.Second,
			5 * time.Second,
			2 * time.Second,
			1 * time.Second,
		})
		assert.Equal(t, 10*time.Second, cb.NextDelay(1))
		assert.Equal(t, 5*time.Second, cb.NextDelay(2))
		assert.Equal(t, 2*time.Second, cb.NextDelay(3))
		assert.Equal(t, 1*time.Second, cb.NextDelay(4))
		assert.Equal(t, 1*time.Second, cb.NextDelay(5))
	})
}