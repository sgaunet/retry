// Command database_wait demonstrates waiting for a database to become
// available using a timeout-based stop condition with linear backoff.
package main

import (
	"context"
	"log"
	"time"

	"github.com/sgaunet/retry/pkg/logger"
	"github.com/sgaunet/retry/pkg/retry"
)

func main() {
	// Wait up to 2 minutes for database to be ready
	condition := retry.NewStopOnTimeout(2 * time.Minute)

	r, err := retry.NewRetry("pg_isready -h localhost -p 5432", condition)
	if err != nil {
		log.Fatal(err)
	}

	// Linear backoff: 2s, 3s, 4s, ... up to 30s
	r.SetBackoffStrategy(retry.NewLinearBackoff(
		2*time.Second,  // base delay
		time.Second,    // increment per attempt
		30*time.Second, // max delay
	))

	appLogger := logger.NewLogger("info")
	err = r.RunWithLogger(context.Background(), appLogger)
	if err != nil {
		log.Fatalf("Database not ready: %v", err)
	}

	log.Println("Database is ready!")
}
