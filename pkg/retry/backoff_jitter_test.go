package retry

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockBackoff is a test helper that returns a predictable delay
type MockBackoff struct {
	Delay time.Duration
}

func (m *MockBackoff) NextDelay(_ int) time.Duration {
	return m.Delay
}

func TestJitterBackoff_NoJitter(t *testing.T) {
	mockStrategy := &MockBackoff{Delay: 1 * time.Second}
	jb := NewJitterBackoff(mockStrategy, 0.0)
	
	// With 0 jitter, should return exact base delay
	for i := 0; i < 10; i++ {
		actual := jb.NextDelay(i)
		assert.Equal(t, 1*time.Second, actual)
	}
}

func TestJitterBackoff_JitterRange(t *testing.T) {
	mockStrategy := &MockBackoff{Delay: 1 * time.Second}
	jb := NewJitterBackoff(mockStrategy, 0.2) // 20% jitter
	
	// With 20% jitter, delay should be between 800ms and 1200ms
	minExpected := 800 * time.Millisecond
	maxExpected := 1200 * time.Millisecond
	
	// Test multiple times to ensure randomness stays within bounds
	for i := 0; i < 100; i++ {
		actual := jb.NextDelay(i)
		assert.GreaterOrEqual(t, actual, minExpected, "Delay should be at least %v", minExpected)
		assert.LessOrEqual(t, actual, maxExpected, "Delay should be at most %v", maxExpected)
	}
}

func TestJitterBackoff_MaxJitter(t *testing.T) {
	mockStrategy := &MockBackoff{Delay: 1 * time.Second}
	jb := NewJitterBackoff(mockStrategy, 1.0) // 100% jitter
	
	// With 100% jitter, delay should be between 0 and 2s
	minExpected := time.Duration(0)
	maxExpected := 2 * time.Second
	
	// Test multiple times
	for i := 0; i < 100; i++ {
		actual := jb.NextDelay(i)
		assert.GreaterOrEqual(t, actual, minExpected)
		assert.LessOrEqual(t, actual, maxExpected)
	}
}

func TestJitterBackoff_ZeroBaseDelay(t *testing.T) {
	mockStrategy := &MockBackoff{Delay: 0}
	jb := NewJitterBackoff(mockStrategy, 0.5)
	
	// With zero base delay, jitter should still return 0
	assert.Equal(t, time.Duration(0), jb.NextDelay(1))
}

func TestJitterBackoff_NilStrategy(t *testing.T) {
	jb := NewJitterBackoff(nil, 0.5)
	
	// With nil strategy, should return 0
	assert.Equal(t, time.Duration(0), jb.NextDelay(1))
}

func TestJitterBackoff_ClampedJitter(t *testing.T) {
	mockStrategy := &MockBackoff{Delay: 1 * time.Second}
	
	// Test jitter clamping for values outside 0-1 range
	t.Run("negative jitter clamped to 0", func(t *testing.T) {
		jb := NewJitterBackoff(mockStrategy, -0.5)
		assert.Equal(t, 0.0, jb.Jitter)
		// Should behave like no jitter
		assert.Equal(t, 1*time.Second, jb.NextDelay(1))
	})
	
	t.Run("jitter above 1 clamped to 1", func(t *testing.T) {
		jb := NewJitterBackoff(mockStrategy, 1.5)
		assert.Equal(t, 1.0, jb.Jitter)
		// Should behave like 100% jitter
		for i := 0; i < 10; i++ {
			actual := jb.NextDelay(i)
			assert.GreaterOrEqual(t, actual, time.Duration(0))
			assert.LessOrEqual(t, actual, 2*time.Second)
		}
	})
}

func TestJitterBackoff_WithRealStrategies(t *testing.T) {
	t.Run("jitter with exponential backoff", func(t *testing.T) {
		expBackoff := NewExponentialBackoff(1*time.Second, 10*time.Second, 2.0)
		jb := NewJitterBackoff(expBackoff, 0.1) // 10% jitter
		
		// Test first few attempts
		for attempt := 1; attempt <= 5; attempt++ {
			baseDelay := expBackoff.NextDelay(attempt)
			jitteredDelay := jb.NextDelay(attempt)
			
			minExpected := time.Duration(float64(baseDelay) * 0.9)
			maxExpected := time.Duration(float64(baseDelay) * 1.1)
			
			assert.GreaterOrEqual(t, jitteredDelay, minExpected)
			assert.LessOrEqual(t, jitteredDelay, maxExpected)
		}
	})
	
	t.Run("jitter with linear backoff", func(t *testing.T) {
		linBackoff := NewLinearBackoff(1*time.Second, 500*time.Millisecond, 5*time.Second)
		jb := NewJitterBackoff(linBackoff, 0.15) // 15% jitter
		
		for attempt := 1; attempt <= 5; attempt++ {
			baseDelay := linBackoff.NextDelay(attempt)
			jitteredDelay := jb.NextDelay(attempt)
			
			minExpected := time.Duration(float64(baseDelay) * 0.85)
			maxExpected := time.Duration(float64(baseDelay) * 1.15)
			
			assert.GreaterOrEqual(t, jitteredDelay, minExpected)
			assert.LessOrEqual(t, jitteredDelay, maxExpected)
		}
	})
}

func TestJitterBackoff_Distribution(t *testing.T) {
	// Test that jitter produces a reasonable distribution
	mockStrategy := &MockBackoff{Delay: 1 * time.Second}
	jb := NewJitterBackoff(mockStrategy, 0.5) // 50% jitter
	
	const samples = 1000
	sum := time.Duration(0)
	minSeen := time.Duration(math.MaxInt64)
	maxSeen := time.Duration(0)
	
	for i := 0; i < samples; i++ {
		delay := jb.NextDelay(1)
		sum += delay
		if delay < minSeen {
			minSeen = delay
		}
		if delay > maxSeen {
			maxSeen = delay
		}
	}
	
	avg := sum / samples
	
	// Average should be close to base delay
	assert.InDelta(t, float64(1*time.Second), float64(avg), float64(100*time.Millisecond),
		"Average jittered delay should be close to base delay")
	
	// Should have seen values across most of the range
	assert.Less(t, minSeen, 700*time.Millisecond, "Should see some low values")
	assert.Greater(t, maxSeen, 1300*time.Millisecond, "Should see some high values")
}