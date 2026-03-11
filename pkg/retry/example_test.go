package retry_test

import (
	"context"
	"fmt"
	"time"

	"github.com/sgaunet/retry/pkg/logger"
	"github.com/sgaunet/retry/pkg/retry"
)

func Example_basic() {
	r, err := retry.NewRetry("echo hello", retry.NewStopOnMaxTries(3))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	r.SetBackoffStrategy(retry.NewFixedBackoff(100 * time.Millisecond))

	err = r.RunWithLogger(context.Background(), logger.NoLogger())
	if err != nil {
		fmt.Println("error:", err)
	}
	// Output:
	// hello
}

func Example_exponentialBackoff() {
	r, _ := retry.NewRetry("echo backoff", retry.NewStopOnMaxTries(3))
	r.SetBackoffStrategy(retry.NewExponentialBackoff(
		100*time.Millisecond, // base delay
		5*time.Second,        // max delay
		2.0,                  // multiplier
	))

	_ = r.RunWithLogger(context.Background(), logger.NoLogger())
	// Output:
	// backoff
}

func Example_compositeCondition() {
	// Stop after 5 tries OR after 10 seconds, whichever comes first
	condition := retry.NewCompositeCondition(
		retry.LogicOR,
		retry.NewStopOnMaxTries(5),
		retry.NewStopOnTimeout(10*time.Second),
	)

	r, _ := retry.NewRetry("echo composite", condition)
	r.SetBackoffStrategy(retry.NewFixedBackoff(100 * time.Millisecond))

	_ = r.RunWithLogger(context.Background(), logger.NoLogger())
	// Output:
	// composite
}

func Example_successCondition() {
	r, _ := retry.NewRetry("echo 'status: ok'", retry.NewStopOnMaxTries(5))

	// Consider it successful if output contains "ok"
	successCond, _ := retry.NewSuccessContains("ok")
	r.SetSuccessConditions([]retry.ConditionRetryer{successCond})

	_ = r.RunWithLogger(context.Background(), logger.NoLogger())
	// Output:
	// status: ok
}

func Example_jitterBackoff() {
	r, _ := retry.NewRetry("echo jitter", retry.NewStopOnMaxTries(3))

	// Wrap exponential backoff with 20% jitter
	backoff := retry.NewJitterBackoff(
		retry.NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 2.0),
		0.2,
	)
	r.SetBackoffStrategy(backoff)

	_ = r.RunWithLogger(context.Background(), logger.NoLogger())
	// Output:
	// jitter
}

func Example_fibonacciBackoff() {
	r, _ := retry.NewRetry("echo fibonacci", retry.NewStopOnMaxTries(3))
	r.SetBackoffStrategy(retry.NewFibonacciBackoff(
		100*time.Millisecond, // base delay
		5*time.Second,        // max delay
	))

	_ = r.RunWithLogger(context.Background(), logger.NoLogger())
	// Output:
	// fibonacci
}

func Example_linearBackoff() {
	r, _ := retry.NewRetry("echo linear", retry.NewStopOnMaxTries(3))
	r.SetBackoffStrategy(retry.NewLinearBackoff(
		100*time.Millisecond, // base delay
		50*time.Millisecond,  // increment per attempt
		5*time.Second,        // max delay
	))

	_ = r.RunWithLogger(context.Background(), logger.NoLogger())
	// Output:
	// linear
}

func Example_customBackoff() {
	r, _ := retry.NewRetry("echo custom", retry.NewStopOnMaxTries(3))
	r.SetBackoffStrategy(retry.NewCustomBackoff([]time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		2 * time.Second,
	}))

	_ = r.RunWithLogger(context.Background(), logger.NoLogger())
	// Output:
	// custom
}

func Example_withContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, _ := retry.NewRetry("echo context", retry.NewStopOnMaxTries(3))
	r.SetBackoffStrategy(retry.NewFixedBackoff(100 * time.Millisecond))

	err := r.RunWithLogger(ctx, logger.NoLogger())
	if err != nil {
		fmt.Println("error:", err)
	}
	// Output:
	// context
}

func Example_withFileLogger() {
	r, _ := retry.NewRetry("echo logged", retry.NewStopOnMaxTries(3))

	// Use a file logger for persistent logging
	appLogger, err := logger.NewFileLogger("info", "/tmp/retry-example.log")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	_ = r.RunWithLogger(context.Background(), appLogger)
	// Output:
	// logged
}
