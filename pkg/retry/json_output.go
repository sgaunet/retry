package retry

import (
	"sync"
	"time"
)

// JSONResult is the top-level JSON output structure.
type JSONResult struct {
	Command             string            `json:"command"`
	Status              string            `json:"status"`
	Attempts            int               `json:"attempts"`
	FinalExitCode       int               `json:"final_exit_code"`
	TotalDuration       string            `json:"total_duration"`
	StopCondition       string            `json:"stop_condition"`
	BackoffStrategy     string            `json:"backoff_strategy"`
	ExecutionDetails    []AttemptDetail   `json:"execution_details"`
	SuccessConditionMet bool              `json:"success_condition_met"`
	ConditionsEvaluated map[string]any    `json:"conditions_evaluated"`
}

// AttemptDetail holds per-attempt data.
type AttemptDetail struct {
	Attempt   int    `json:"attempt"`
	ExitCode  int    `json:"exit_code"`
	Duration  string `json:"duration"`
	Timestamp string `json:"timestamp"`
}

// AttemptCollector accumulates per-attempt data during retry execution.
type AttemptCollector struct {
	mu       sync.Mutex
	attempts []AttemptDetail
	start    time.Time
}

// NewAttemptCollector creates a new AttemptCollector.
func NewAttemptCollector() *AttemptCollector {
	return &AttemptCollector{
		start: time.Now(),
	}
}

// RecordAttempt records data for a single attempt.
func (c *AttemptCollector) RecordAttempt(attempt, exitCode int, duration time.Duration, startTime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts = append(c.attempts, AttemptDetail{
		Attempt:   attempt,
		ExitCode:  exitCode,
		Duration:  duration.String(),
		Timestamp: startTime.Format(time.RFC3339),
	})
}

// GetAttempts returns a copy of all recorded attempts.
func (c *AttemptCollector) GetAttempts() []AttemptDetail {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]AttemptDetail, len(c.attempts))
	copy(result, c.attempts)
	return result
}

// TotalDuration returns the total duration since the collector was created.
func (c *AttemptCollector) TotalDuration() time.Duration {
	return time.Since(c.start)
}
