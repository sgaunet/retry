// Command api_check demonstrates retrying an API health check with
// composite stop conditions and jitter backoff.
package main

import (
	"context"
	"log"
	"time"

	"github.com/sgaunet/retry/pkg/logger"
	"github.com/sgaunet/retry/pkg/retry"
)

func main() {
	// Stop after 10 tries OR after 5 minutes, whichever comes first
	condition := retry.NewCompositeCondition(
		retry.LogicOR,
		retry.NewStopOnMaxTries(10),
		retry.NewStopOnTimeout(5*time.Minute),
	)

	r, err := retry.NewRetry("curl -sf https://api.example.com/health", condition)
	if err != nil {
		log.Fatal(err)
	}

	// Consider it successful if output contains "healthy"
	successCond, _ := retry.NewSuccessContains("healthy")
	r.SetSuccessConditions([]retry.ConditionRetryer{successCond})

	// Exponential backoff with 20% jitter to avoid thundering herd
	backoff := retry.NewJitterBackoff(
		retry.NewExponentialBackoff(2*time.Second, 2*time.Minute, 2.0),
		0.2,
	)
	r.SetBackoffStrategy(backoff)

	// Run with context for graceful cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	appLogger := logger.NewLogger("info")
	err = r.RunWithLogger(ctx, appLogger)
	if err != nil {
		log.Fatalf("API health check failed: %v", err)
	}

	log.Println("API is healthy!")
}
