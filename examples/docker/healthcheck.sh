#!/bin/bash
# Docker healthcheck script using retry
#
# Use this as a HEALTHCHECK command in your Dockerfile:
#   HEALTHCHECK --interval=10s --timeout=30s --retries=1 \
#     CMD ["/usr/local/bin/healthcheck.sh"]
#
# retry handles the retry logic internally, so Docker only needs
# to run this script once per health check interval.

set -e

# Wait for the application to respond with "healthy" status
# Strategy: Linear backoff (2s, 3s, 4s, ...) up to 10s, timeout after 25s
retry --max-tries 0 \
      --timeout 25s \
      --backoff linear \
      --base-delay 2s \
      --increment 1s \
      --max-delay 10s \
      --success-contains "healthy" \
      --quiet \
      "curl -sf http://localhost:8080/health"
