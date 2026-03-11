#!/bin/bash
# Wait for a service to start and become healthy
#
# Useful as an entrypoint wrapper or in startup scripts
# to ensure dependencies are ready before the main process starts.
#
# Usage:
#   ./service-startup-wait.sh http://localhost:8080/health
#   ./service-startup-wait.sh http://redis:6379 http://postgres:5432

set -e

if [ $# -eq 0 ]; then
  echo "Usage: $0 <url> [url...]"
  echo "Example: $0 http://localhost:8080/health http://db:5432"
  exit 1
fi

for url in "$@"; do
  echo "Waiting for ${url}..."

  # Strategy: Exponential backoff, timeout after 2 minutes per service
  retry --max-tries 0 \
        --timeout 2m \
        --backoff exponential \
        --base-delay 1s \
        --max-delay 15s \
        --quiet \
        "curl -sf ${url}"

  echo "${url} is ready."
done

echo "All services are ready!"
