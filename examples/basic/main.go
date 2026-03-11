// Command basic demonstrates simple retry usage with exponential backoff.
package main

import (
	"context"
	"log"
	"time"

	"github.com/sgaunet/retry/pkg/logger"
	"github.com/sgaunet/retry/pkg/retry"
)

func main() {
	// Simple retry with max 3 attempts
	r, err := retry.NewRetry("echo 'Hello World'", retry.NewStopOnMaxTries(3))
	if err != nil {
		log.Fatal(err)
	}

	// Add exponential backoff between retries
	r.SetBackoffStrategy(retry.NewExponentialBackoff(
		time.Second,      // base delay
		30*time.Second,   // max delay
		2.0,              // multiplier (1s, 2s, 4s, 8s, ...)
	))

	// Run with an info-level logger
	appLogger := logger.NewLogger("info")
	err = r.RunWithLogger(context.Background(), appLogger)
	if err != nil {
		log.Printf("Command failed after retries: %v", err)
	}
}
