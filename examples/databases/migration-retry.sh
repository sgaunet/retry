#!/bin/bash
# Database migration with retry and proper error handling
#
# This script waits for the database, then runs migrations with retry.
# It uses --fail-if-contains to stop immediately on fatal errors
# that would not resolve with retries.
#
# Usage:
#   ./migration-retry.sh
#   DB_HOST=mydb MIGRATIONS_PATH=./db/migrations ./migration-retry.sh

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-myuser}"
DB_PASS="${DB_PASS:-mypassword}"
DB_NAME="${DB_NAME:-mydb}"
MIGRATIONS_PATH="${MIGRATIONS_PATH:-./migrations}"

DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Step 1: Wait for database to be ready
echo "Waiting for database..."
retry --max-tries 20 \
      --backoff exponential \
      --base-delay 2s \
      --max-delay 30s \
      --quiet \
      "pg_isready -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER}"

echo "Database is ready."

# Step 2: Run migrations with retry
# Strategy: Fixed delay, max 3 attempts
# --fail-if-contains "FATAL" stops immediately on unrecoverable errors
# like syntax errors in migration files
echo "Running migrations..."
retry --max-tries 3 \
      --backoff fixed \
      --delay 5s \
      --fail-if-contains "FATAL" \
      --log-level info \
      "migrate -path ${MIGRATIONS_PATH} -database ${DATABASE_URL} up"

echo "Migrations completed successfully."
