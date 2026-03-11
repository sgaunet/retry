# retry Examples

This directory contains real-world examples for using `retry` in various scenarios.

## CLI Examples (Shell Scripts & Configs)

### CI/CD Integration
- [GitHub Actions](ci-cd/github-actions.yml) - Deploy with retry in GitHub Actions
- [GitLab CI](ci-cd/gitlab-ci.yml) - GitLab CI pipeline with retry stages

### Docker & Containers
- [Health Check](docker/healthcheck.sh) - Docker container health check script
- [Dockerfile](docker/Dockerfile.example) - Multi-stage Dockerfile with retry
- [Docker Compose](docker/docker-compose.yml) - Service dependency handling

### Databases
- [Migration Retry](databases/migration-retry.sh) - Database migration with retry and error handling
- [Connection Wait](databases/connection-wait.sh) - Wait for database availability

### Networking
- [API Endpoint Check](networking/api-endpoint-check.sh) - API health check with multiple conditions
- [Download with Retry](networking/download-with-retry.sh) - Reliable file downloads

### Kubernetes
- [Init Container](kubernetes/init-container.yaml) - Wait for dependencies before starting
- [Job with Retry](kubernetes/job-with-retry.yaml) - Kubernetes Job with retry sidecar

### Testing
- [Flaky Tests](testing/flaky-tests.sh) - Handle flaky tests intelligently
- [E2E Tests](testing/e2e-with-retry.sh) - End-to-end test retry strategies

### Services
- [Service Startup Wait](services/service-startup-wait.sh) - Wait for service availability
- [Dependency Check](services/dependency-check.sh) - Multi-service dependency verification

## Go Library Examples

For programmatic usage of the retry package in Go applications:

- [Basic](basic/) - Simple retry with exponential backoff
- [API Check](api_check/) - API health check with composite conditions and jitter
- [Database Wait](database_wait/) - Wait for database with linear backoff

## Quick Reference

### Common Patterns

```bash
# Simple retry (3 attempts, no delay)
retry "your-command"

# Fixed delay between retries
retry --max-tries 5 --delay 2s "your-command"

# Exponential backoff (recommended for network operations)
retry --backoff exponential --base-delay 1s --max-delay 30s "your-command"

# Timeout-based (retry until timeout, no max tries limit)
retry --max-tries 0 --timeout 5m --backoff exponential "your-command"

# Success based on output content
retry --success-contains "healthy" --timeout 2m "curl http://localhost:8080/health"

# Fail fast on fatal errors
retry --fail-if-contains "FATAL" --max-tries 10 "your-command"
```

### Environment Variables

All flags can be set via environment variables with the `RETRY_` prefix:

```bash
export RETRY_MAX_TRIES=5
export RETRY_BACKOFF=exponential
export RETRY_BASE_DELAY=1s
export RETRY_MAX_DELAY=30s
export RETRY_TIMEOUT=5m
retry "your-command"
```
