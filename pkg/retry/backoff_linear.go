package retry

import (
	"time"
)

// LinearBackoff implements a linear backoff strategy.
type LinearBackoff struct {
	BaseDelay time.Duration
	Increment time.Duration
	MaxDelay  time.Duration
}

// NewLinearBackoff creates a new LinearBackoff instance.
func NewLinearBackoff(baseDelay, increment, maxDelay time.Duration) *LinearBackoff {
	return &LinearBackoff{
		BaseDelay: baseDelay,
		Increment: increment,
		MaxDelay:  maxDelay,
	}
}

// NextDelay calculates the next delay using linear backoff.
// Formula: delay = baseDelay + (attempt-1) * increment.
func (l *LinearBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return l.BaseDelay
	}
	
	delay := l.BaseDelay + time.Duration(attempt-1)*l.Increment
	
	// Cap at MaxDelay if specified
	if l.MaxDelay > 0 && delay > l.MaxDelay {
		return l.MaxDelay
	}
	
	return delay
}