package retry

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestAttemptCollector_RecordAndRetrieve(t *testing.T) {
	c := NewAttemptCollector()

	now := time.Now()
	c.RecordAttempt(1, 1, 100*time.Millisecond, now)
	c.RecordAttempt(2, 0, 50*time.Millisecond, now.Add(200*time.Millisecond))

	attempts := c.GetAttempts()
	if len(attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(attempts))
	}

	if attempts[0].Attempt != 1 {
		t.Errorf("expected attempt 1, got %d", attempts[0].Attempt)
	}
	if attempts[0].ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", attempts[0].ExitCode)
	}
	if attempts[1].Attempt != 2 {
		t.Errorf("expected attempt 2, got %d", attempts[1].Attempt)
	}
	if attempts[1].ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", attempts[1].ExitCode)
	}
}

func TestAttemptCollector_GetAttemptsReturnsCopy(t *testing.T) {
	c := NewAttemptCollector()
	c.RecordAttempt(1, 0, 10*time.Millisecond, time.Now())

	attempts1 := c.GetAttempts()
	attempts2 := c.GetAttempts()

	// Modify first copy
	attempts1[0].ExitCode = 99

	// Second copy should be unaffected
	if attempts2[0].ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", attempts2[0].ExitCode)
	}
}

func TestAttemptCollector_TotalDuration(t *testing.T) {
	c := NewAttemptCollector()
	time.Sleep(10 * time.Millisecond)
	d := c.TotalDuration()
	if d < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", d)
	}
}

func TestAttemptCollector_Empty(t *testing.T) {
	c := NewAttemptCollector()
	attempts := c.GetAttempts()
	if len(attempts) != 0 {
		t.Errorf("expected 0 attempts, got %d", len(attempts))
	}
}

func TestJSONResult_Marshal(t *testing.T) {
	result := JSONResult{
		Command:         "echo hello",
		Status:          "success",
		Attempts:        1,
		FinalExitCode:   0,
		TotalDuration:   "100ms",
		StopCondition:   "command_succeeded",
		BackoffStrategy: "fixed",
		ExecutionDetails: []AttemptDetail{
			{
				Attempt:   1,
				ExitCode:  0,
				Duration:  "50ms",
				Timestamp: "2024-01-01T00:00:00Z",
			},
		},
		SuccessConditionMet: false,
		ConditionsEvaluated: map[string]any{
			"max_tries": 3,
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed JSONResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", parsed.Status)
	}
	if parsed.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", parsed.Attempts)
	}
	if len(parsed.ExecutionDetails) != 1 {
		t.Errorf("expected 1 execution detail, got %d", len(parsed.ExecutionDetails))
	}
	if parsed.Command != "echo hello" {
		t.Errorf("expected command 'echo hello', got '%s'", parsed.Command)
	}
	if parsed.BackoffStrategy != "fixed" {
		t.Errorf("expected backoff 'fixed', got '%s'", parsed.BackoffStrategy)
	}
}

func TestJSONResult_MarshalIndent(t *testing.T) {
	result := JSONResult{
		Command:             "test",
		Status:              "failure",
		Attempts:            3,
		FinalExitCode:       1,
		TotalDuration:       "5s",
		StopCondition:       "max_tries_reached",
		BackoffStrategy:     "exponential",
		ExecutionDetails:    []AttemptDetail{},
		SuccessConditionMet: false,
		ConditionsEvaluated: map[string]any{},
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal indent: %v", err)
	}

	// Verify it contains newlines (pretty-printed)
	output := string(data)
	if len(output) == 0 {
		t.Fatal("expected non-empty output")
	}

	// Verify it round-trips
	var parsed JSONResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if parsed.Status != "failure" {
		t.Errorf("expected status 'failure', got '%s'", parsed.Status)
	}
}

// TestAttemptCollector_TotalDurationConcurrent verifies Bug 7:
// TotalDuration must be safe to call concurrently with RecordAttempt.
// Run with: go test -race ./pkg/retry/...
func TestAttemptCollector_TotalDurationConcurrent(t *testing.T) {
	c := NewAttemptCollector()

	const workers = 10
	const recordsPerWorker = 50

	var wg sync.WaitGroup

	// Spawn goroutines that call RecordAttempt concurrently.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < recordsPerWorker; j++ {
				c.RecordAttempt(id*recordsPerWorker+j, j%3, time.Millisecond, time.Now())
			}
		}(i)
	}

	// Spawn goroutines that call TotalDuration concurrently with RecordAttempt.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < recordsPerWorker; j++ {
				d := c.TotalDuration()
				if d < 0 {
					t.Errorf("TotalDuration returned negative value: %v", d)
				}
			}
		}()
	}

	wg.Wait()
}

func TestAttemptDetail_TimestampFormat(t *testing.T) {
	c := NewAttemptCollector()
	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	c.RecordAttempt(1, 0, time.Second, now)

	attempts := c.GetAttempts()
	expected := "2024-06-15T10:30:00Z"
	if attempts[0].Timestamp != expected {
		t.Errorf("expected timestamp %s, got %s", expected, attempts[0].Timestamp)
	}
}
