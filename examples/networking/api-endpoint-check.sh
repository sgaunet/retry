#!/bin/bash
# API endpoint health check with multiple conditions
#
# Demonstrates combining stop conditions and success conditions
# for robust API monitoring.
#
# Usage:
#   ./api-endpoint-check.sh
#   API_URL=https://api.example.com/health ./api-endpoint-check.sh

set -e

API_URL="${API_URL:-https://api.example.com/health}"

echo "Checking API endpoint: ${API_URL}"

# Strategy: Fibonacci backoff (1s, 1s, 2s, 3s, 5s, 8s, ...)
# Combines max tries (10) with timeout (5m) using OR logic
# Considers success when output matches HTTP 2xx or 3xx status
retry --max-tries 10 \
      --timeout 5m \
      --backoff fibonacci \
      --base-delay 1s \
      --max-delay 1m \
      --success-regex "HTTP/[12].[01] [23][0-9][0-9]" \
      "curl -sI ${API_URL}"

echo "API endpoint is healthy!"
