#!/bin/bash
# Wait for database to become available
#
# Use this script in CI/CD pipelines, Docker entrypoints, or
# init containers to wait for the database before starting your app.
#
# Usage:
#   ./connection-wait.sh              # Uses defaults
#   DB_HOST=mydb DB_PORT=5432 ./connection-wait.sh

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"

echo "Waiting for database at ${DB_HOST}:${DB_PORT}..."

# Strategy: Exponential backoff starting at 1s, capped at 15s
# Timeout after 2 minutes total
retry --max-tries 0 \
      --timeout 2m \
      --backoff exponential \
      --base-delay 1s \
      --max-delay 15s \
      --quiet \
      "pg_isready -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER}"

echo "Database is ready!"
