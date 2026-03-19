package retry

import (
	"math"
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

// maxSafeFibMultiplier is the largest Fibonacci number that can be multiplied by
// a 1-nanosecond base delay without overflowing time.Duration (int64).
// time.Duration max is math.MaxInt64 nanoseconds ≈ 9.2×10^18 ns.
// We cap the multiplier conservatively so that fibMultiplier * baseDelay never wraps.
const maxSafeFibMultiplier = math.MaxInt32 // ~2.1×10^9, safe for any base delay ≥ 1ns

// NextDelay calculates the next delay using Fibonacci sequence.
// The delay is baseDelay multiplied by the Fibonacci number at the attempt position.
// For attempts beyond the int overflow boundary (>93), the Fibonacci multiplier is
// capped so the result never becomes negative.
func (f *FibonacciBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return f.BaseDelay
	}

	// Ensure we have enough Fibonacci numbers calculated.
	for len(f.sequence) < attempt {
		prev1 := f.sequence[len(f.sequence)-1]
		prev2 := f.sequence[len(f.sequence)-2]
		nextFib := prev1 + prev2
		// Detect integer overflow: result wrapped or either addend was already capped.
		if nextFib < prev1 || prev1 == maxSafeFibMultiplier {
			nextFib = maxSafeFibMultiplier
		}
		f.sequence = append(f.sequence, nextFib)
	}

	fibMultiplier := min(f.sequence[attempt-1], maxSafeFibMultiplier)

	delay := time.Duration(fibMultiplier) * f.BaseDelay

	// Negative result means duration overflow despite the multiplier cap
	// (e.g. very large BaseDelay). Return MaxDelay or BaseDelay as fallback.
	if delay < 0 {
		if f.MaxDelay > 0 {
			return f.MaxDelay
		}
		return f.BaseDelay
	}

	// Cap at MaxDelay if specified.
	if f.MaxDelay > 0 && delay > f.MaxDelay {
		return f.MaxDelay
	}

	return delay
}