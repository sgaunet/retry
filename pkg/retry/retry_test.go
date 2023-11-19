package retry

import (
	"testing"
	"time"
)

func Test_retry_SetSleep(t *testing.T) {
	r, err := NewRetry("", NewStopOnMaxTries(3))
	if err != nil {
		t.Errorf("Expected no error")
	}
	r.SetSleep(func() { time.Sleep(5 * time.Second) })
	if r.sleep == nil {
		t.Errorf("Expected sleep function to be set")
	}
}
