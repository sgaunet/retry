#!/bin/bash
# Reliable file download with retry
#
# Downloads a file with exponential backoff and jitter.
# Jitter prevents multiple clients from hammering the server
# simultaneously after a transient failure.
#
# Usage:
#   ./download-with-retry.sh https://example.com/large-file.tar.gz output.tar.gz

set -e

URL="${1:?Usage: $0 <url> <output-file>}"
OUTPUT="${2:?Usage: $0 <url> <output-file>}"

echo "Downloading ${URL} -> ${OUTPUT}"

# Strategy: Exponential backoff with 30% jitter
# curl -C - enables resume on partial downloads
retry --max-tries 5 \
      --backoff exponential \
      --base-delay 3s \
      --max-delay 1m \
      --jitter 0.3 \
      "curl -fSL -C - -o ${OUTPUT} ${URL}"

echo "Download complete: ${OUTPUT}"
