package retry

import (
	"math"
	"time"
)

// BackoffStrategy defines the interface for different backoff strategies.
type BackoffStrategy interface {
	NextDelay(attempt int) time.Duration
}

// FixedBackoff implements a fixed delay strategy.
type FixedBackoff struct {
	Delay time.Duration
}

// NewFixedBackoff creates a new FixedBackoff instance.
func NewFixedBackoff(delay time.Duration) *FixedBackoff {
	return &FixedBackoff{Delay: delay}
}

// NextDelay returns the fixed delay regardless of attempt number.
func (f *FixedBackoff) NextDelay(_ int) time.Duration {
	return f.Delay
}

// ExponentialBackoff implements an exponential backoff strategy.
type ExponentialBackoff struct {
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// NewExponentialBackoff creates a new ExponentialBackoff instance.
func NewExponentialBackoff(baseDelay, maxDelay time.Duration, multiplier float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		BaseDelay:  baseDelay,
		MaxDelay:   maxDelay,
		Multiplier: multiplier,
	}
}

// NextDelay calculates the next delay using exponential backoff.
func (e *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return e.BaseDelay
	}
	
	delay := float64(e.BaseDelay) * math.Pow(e.Multiplier, float64(attempt-1))
	
	// Cap at MaxDelay
	if delay > float64(e.MaxDelay) {
		return e.MaxDelay
	}
	
	// Check for overflow
	if delay > float64(math.MaxInt64) {
		return e.MaxDelay
	}
	
	return time.Duration(delay)
}