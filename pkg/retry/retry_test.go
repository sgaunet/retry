package retry

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

func Test_retry_SetSleep(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := NewRetry("", NewStopOnMaxTries(3))
	if err != nil {
		t.Errorf("Expected no error")
	}
	r.SetSleep(func() { time.Sleep(5 * time.Second) })
	if r.sleep == nil {
		t.Errorf("Expected sleep function to be set")
	}
}
