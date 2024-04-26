package retry_test

import (
	"testing"
	"time"

	"github.com/sgaunet/retry/pkg/retry"
	"go.uber.org/goleak"
)

func TestEmptyCommand(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := retry.NewRetry("", retry.NewStopOnMaxTries(3))
	if err != nil {
		t.Errorf("Expected no error")
	}
	err = r.Run()
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestRetryWithSleep(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := retry.NewRetry("ls -l '/sdfsdfqsdbj'", retry.NewStopOnMaxTries(3))
	if err != nil {
		t.Errorf("Expected no error")
	}
	r.SetSleep(func() {
		time.Sleep(1 * time.Second)
	})
	startTime := time.Now()
	err = r.Run()
	endTime := time.Now()
	if err == nil {
		t.Errorf("Expected error")
	}
	if endTime.Sub(startTime) < 3*time.Second {
		t.Errorf("Expected at least 3 seconds, got %v", endTime.Sub(startTime))
	}
}

func TestRetryWithSleep2(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := retry.NewRetry("sleep 4", retry.NewStopOnMaxExecTime(5*time.Millisecond))
	if err != nil {
		t.Errorf("Expected no error")
	}
	startTime := time.Now()
	err = r.Run()
	endTime := time.Now()
	if err == nil {
		t.Errorf("Expected error")
	}
	if endTime.Sub(startTime) < 3*time.Second {
		t.Errorf("Expected at least 3 seconds, got %v", endTime.Sub(startTime))
	}
}
