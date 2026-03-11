#!/bin/bash
# End-to-end test retry strategy
#
# E2E tests often depend on external services and infrastructure.
# This script demonstrates a two-phase approach:
# 1. Wait for the test environment to be ready
# 2. Run E2E tests with retry for transient failures
#
# Usage:
#   ./e2e-with-retry.sh
#   APP_URL=https://staging.example.com E2E_CMD="npx cypress run" ./e2e-with-retry.sh

set -e

APP_URL="${APP_URL:-http://localhost:3000}"
E2E_CMD="${E2E_CMD:-npx playwright test}"

# Phase 1: Wait for the application to be ready
echo "Waiting for application at ${APP_URL}..."
retry --max-tries 0 \
      --timeout 3m \
      --backoff exponential \
      --base-delay 2s \
      --max-delay 15s \
      --quiet \
      "curl -sf ${APP_URL}/health"

echo "Application is ready."

# Phase 2: Run E2E tests with retry
# Strategy: Longer delay between retries (30s) to let infrastructure stabilize
echo "Running E2E tests..."
retry --max-tries 3 \
      --backoff fixed \
      --delay 30s \
      --fail-if-contains "Error: page crashed" \
      --log-level info \
      "${E2E_CMD}"

echo "E2E tests passed!"
