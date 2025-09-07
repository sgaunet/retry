package retry

import (
	"time"
)

// CustomBackoff implements a custom delay sequence backoff strategy.
type CustomBackoff struct {
	Delays []time.Duration
}

// NewCustomBackoff creates a new CustomBackoff instance with specified delays.
func NewCustomBackoff(delays []time.Duration) *CustomBackoff {
	return &CustomBackoff{
		Delays: delays,
	}
}

// NextDelay returns the delay for the given attempt from the custom sequence.
// If attempt exceeds the number of defined delays, it returns the last delay.
func (c *CustomBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 || len(c.Delays) == 0 {
		return 0
	}
	
	// Use the delay at attempt-1 index, or the last delay if we've exceeded the list
	index := attempt - 1
	if index >= len(c.Delays) {
		index = len(c.Delays) - 1
	}
	
	return c.Delays[index]
}