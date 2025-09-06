package retry

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

func Test_retry_SetBackoffStrategy(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := NewRetry("", NewStopOnMaxTries(3))
	if err != nil {
		t.Errorf("Expected no error")
	}
	strategy := NewFixedBackoff(5 * time.Second)
	r.SetBackoffStrategy(strategy)
	if r.backoff == nil {
		t.Errorf("Expected backoff strategy to be set")
	}
}
