package retry

import (
	"crypto/rand"
	"math/big"
	"time"
)

// JitterBackoff wraps another BackoffStrategy and adds random jitter.
type JitterBackoff struct {
	Strategy BackoffStrategy
	Jitter   float64 // Jitter percentage (0.0 to 1.0)
}

// NewJitterBackoff creates a new JitterBackoff that wraps another strategy.
func NewJitterBackoff(strategy BackoffStrategy, jitter float64) *JitterBackoff {
	// Ensure jitter is within valid range
	if jitter < 0 {
		jitter = 0
	} else if jitter > 1 {
		jitter = 1
	}
	
	return &JitterBackoff{
		Strategy: strategy,
		Jitter:   jitter,
	}
}

// NextDelay returns the delay from the wrapped strategy with added jitter.
// Jitter adds randomness of ±jitter% to the base delay.
func (j *JitterBackoff) NextDelay(attempt int) time.Duration {
	if j.Strategy == nil {
		return 0
	}
	
	baseDelay := j.Strategy.NextDelay(attempt)
	if baseDelay == 0 || j.Jitter == 0 {
		return baseDelay
	}
	
	// Calculate jitter range
	jitterRange := float64(baseDelay) * j.Jitter
	
	// Random value between -jitterRange and +jitterRange
	// Using crypto/rand for secure randomness
	maxInt := big.NewInt(1<<53 - 1) // Max safe integer for float64 mantissa
	n, err := rand.Int(rand.Reader, maxInt)
	if err != nil {
		// Fallback to no jitter on error
		return baseDelay
	}
	// Convert to float64 in range [0, 1), then to [-1, 1)
	randomFloat := float64(n.Int64()) / float64(maxInt.Int64())
	jitterValue := (randomFloat*2 - 1) * jitterRange
	
	// Apply jitter to base delay
	finalDelay := float64(baseDelay) + jitterValue
	
	// Ensure delay is not negative
	if finalDelay < 0 {
		finalDelay = 0
	}
	
	return time.Duration(finalDelay)
}