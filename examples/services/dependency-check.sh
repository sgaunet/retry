#!/bin/bash
# Multi-service dependency verification
#
# Checks that all required services are available before
# starting the main application. Each service gets its own
# retry configuration based on expected startup time.
#
# Usage:
#   ./dependency-check.sh

set -e

echo "Verifying service dependencies..."

# Database: may take longer to start, use longer timeout
echo "[1/3] Checking database..."
retry --max-tries 0 \
      --timeout 3m \
      --backoff exponential \
      --base-delay 2s \
      --max-delay 20s \
      --quiet \
      "pg_isready -h ${DB_HOST:-localhost} -p ${DB_PORT:-5432}"
echo "  Database: OK"

# Redis: starts quickly, short timeout
echo "[2/3] Checking Redis..."
retry --max-tries 10 \
      --backoff fixed \
      --delay 1s \
      --quiet \
      "redis-cli -h ${REDIS_HOST:-localhost} -p ${REDIS_PORT:-6379} ping"
echo "  Redis: OK"

# External API: may be rate-limited, use jitter
echo "[3/3] Checking external API..."
retry --max-tries 5 \
      --backoff exponential \
      --base-delay 3s \
      --max-delay 30s \
      --jitter 0.3 \
      --success-contains "ok" \
      --quiet \
      "curl -sf ${API_URL:-https://api.example.com/status}"
echo "  External API: OK"

echo ""
echo "All dependencies verified. Starting application..."
