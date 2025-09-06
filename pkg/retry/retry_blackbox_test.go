package retry_test

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sgaunet/retry/pkg/retry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

var nologger *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

func TestEmptyCommand(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := retry.NewRetry("", retry.NewStopOnMaxTries(3))
	if err != nil {
		t.Errorf("Expected no error")
	}
	err = r.Run(nologger)
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
	r.SetBackoffStrategy(retry.NewFixedBackoff(1 * time.Second))
	startTime := time.Now()
	err = r.Run(nologger)
	endTime := time.Now()
	if err == nil {
		t.Errorf("Expected error")
	}
	if endTime.Sub(startTime) < 2*time.Second {
		t.Errorf("Expected at least 2 seconds, got %v", endTime.Sub(startTime))
	}
}

func TestRetryWithSleep2(t *testing.T) {
	defer goleak.VerifyNone(t)
	r, err := retry.NewRetry("bash -c 'sleep 4'", retry.NewStopOnMaxExecTime(50*time.Millisecond))
	assert.Nil(t, err)
	startTime := time.Now()
	err = r.Run(nologger)
	endTime := time.Now()
	assert.NotNil(t, err, "command should be stopped by max exec time")
	assert.GreaterOrEqual(t, endTime.Sub(startTime).Milliseconds(), int64(50), "Expected at least 50 Milliseconds")
}
