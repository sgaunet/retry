#!/bin/bash
# Handle flaky tests intelligently
#
# Uses --retry-if-contains to only retry on known transient errors
# and --fail-if-contains to stop immediately on real test failures.
#
# Usage:
#   ./flaky-tests.sh                    # Run with defaults
#   TEST_CMD="npm test" ./flaky-tests.sh  # Custom test command

set -e

TEST_CMD="${TEST_CMD:-go test ./...}"

echo "Running tests with intelligent retry..."

# Strategy: Fixed 2s delay between retries
# Only retry if the failure looks transient (connection issues, timeouts)
# Stop immediately on real assertion failures
retry --max-tries 3 \
      --backoff fixed \
      --delay 2s \
      --retry-if-contains "connection refused" \
      --retry-if-contains "timeout" \
      --fail-if-contains "FAIL" \
      --log-level info \
      "${TEST_CMD}"

echo "Tests passed!"
