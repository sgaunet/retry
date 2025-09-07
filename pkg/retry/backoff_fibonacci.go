package retry

import (
	"time"
)

// FibonacciBackoff implements a Fibonacci sequence based backoff strategy.
type FibonacciBackoff struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
	sequence  []int
}

// NewFibonacciBackoff creates a new FibonacciBackoff instance.
func NewFibonacciBackoff(baseDelay, maxDelay time.Duration) *FibonacciBackoff {
	return &FibonacciBackoff{
		BaseDelay: baseDelay,
		MaxDelay:  maxDelay,
		sequence:  []int{1, 1}, // Start with [1, 1] for Fibonacci sequence
	}
}

// NextDelay calculates the next delay using Fibonacci sequence.
// The delay is baseDelay multiplied by the Fibonacci number at the attempt position.
func (f *FibonacciBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return f.BaseDelay
	}
	
	// Ensure we have enough Fibonacci numbers calculated
	for len(f.sequence) < attempt {
		nextFib := f.sequence[len(f.sequence)-1] + f.sequence[len(f.sequence)-2]
		f.sequence = append(f.sequence, nextFib)
	}
	
	// Calculate delay based on Fibonacci number
	fibMultiplier := f.sequence[attempt-1]
	delay := time.Duration(fibMultiplier) * f.BaseDelay
	
	// Cap at MaxDelay if specified
	if f.MaxDelay > 0 && delay > f.MaxDelay {
		return f.MaxDelay
	}
	
	return delay
}