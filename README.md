![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/retry/total)
![GitHub Release](https://img.shields.io/github/v/release/sgaunet/retry)
![Test Coverage](https://raw.githubusercontent.com/wiki/sgaunet/retry/coverage-badge.svg)
[![Linter](https://github.com/sgaunet/retry/actions/workflows/linter.yml/badge.svg)](https://github.com/sgaunet/retry/actions/workflows/linter.yml)
[![Vulnerability Scan](https://github.com/sgaunet/retry/actions/workflows/vulnerability-scan.yml/badge.svg)](https://github.com/sgaunet/retry/actions/workflows/vulnerability-scan.yml)
[![Generate coverage badges](https://github.com/sgaunet/retry/actions/workflows/coverage.yml/badge.svg)](https://github.com/sgaunet/retry/actions/workflows/coverage.yml)
[![Snapshot](https://github.com/sgaunet/retry/actions/workflows/snapshot.yml/badge.svg)](https://github.com/sgaunet/retry/actions/workflows/snapshot.yml)
[![Release](https://github.com/sgaunet/retry/actions/workflows/release.yml/badge.svg)](https://github.com/sgaunet/retry/actions/workflows/release.yml)

# retry

retry command will execute X times a failed command until it's successful. Supports both fixed delays and exponential backoff strategies. Useful for flaky tests, waiting for services to become available, or handling transient failures.

## Features

- **Fixed delay**: Traditional constant delay between retries
- **Exponential backoff**: Smart retry strategy that increases delays exponentially
- **Configurable**: Customize retry count, delays, multipliers, and maximum delays
- **Backward compatible**: Existing scripts continue to work unchanged
- **Environment variables**: Configure via environment variables

# Getting started

## Basic Usage

```bash
# Basic retry with default settings (3 attempts, fixed delay)
retry "flaky-command"

# Custom retry count and fixed delay
retry --max-tries 5 --delay 2s "curl https://api.example.com"

# Exponential backoff (recommended for network operations)
retry --backoff exponential --base-delay 1s --max-delay 30s "curl https://api.example.com"
```

## Exponential Backoff Examples

```bash
# Basic exponential backoff (1s, 2s, 4s, 8s, ...)
retry --backoff exponential "make test"

# Custom exponential backoff with shorter delays  
retry --backoff exp --base-delay 100ms --multiplier 1.5 --max-delay 10s "flaky-service-check"

# Short form flags
retry -B exp -b 500ms -M 1m -t 10 "network-dependent-command"
```

## All Available Options

```
$ retry --help
retry is a CLI tool that executes commands repeatedly until they succeed
or a specified limit is reached. This is useful for handling flaky tests,
waiting for services to become available, or dealing with transient failures.

The command to retry should be provided as a positional argument and quoted
if it contains spaces or special characters.

Usage:
  retry [flags] "command"
  retry [command]

Examples:
  (110 lines of examples omitted - run "retry --help" to see them)

Available Commands:
  completion  Generate completion script
  config      Manage retry configuration files
  help        Help about any command
  version     Print the version number

Flags:
  -B, --backoff string                  backoff strategy (fixed, exponential, linear, fibonacci, custom) (default "fixed")
  -b, --base-delay string               base delay for backoff strategies (default "1s")
      --condition-logic string          logic for multiple conditions (AND or OR) (default "OR")
  -c, --config string                   path to config file
  -d, --delay string                    delay between retries (e.g., 1s, 500ms, 2m) (default "0s")
      --delays string                   comma-separated custom delays (e.g., 1s,2s,5s,10s)
      --fail-if-contains string         fail immediately if pattern found
  -h, --help                            help for retry
      --increment string                increment for linear backoff (default "500ms")
  -j, --jitter float                    jitter percentage (0.0-1.0) to add randomness
      --json                            output results as JSON
      --json-pretty                     output results as pretty-printed JSON
      --list-policies                   list all available retry policies
  -f, --log-file string                 write logs to file
  -l, --log-level string                set log level (error, warn, info, debug) (default "info")
  -M, --max-delay string                maximum delay cap for backoff strategies (default "5m")
  -t, --max-tries uint                  maximum number of retry attempts (0 for infinite) (default 3)
      --multiplier float                multiplier for exponential backoff (default 2)
  -P, --policy string                   use a named retry policy preset
      --profile string                  named profile from config file
  -q, --quiet                           minimal output (only show final result)
      --quiet-retries                   only show command output on final attempt
      --retry-if-contains string        retry if output contains pattern
      --retry-on-exit string            only retry on specific exit codes (comma-separated)
      --retry-regex string              retry if output matches regex
      --show-policy string              show details for a specific policy
      --stop-at string                  stop at specific time (HH:MM format)
      --stop-on-exit string             stop on specific exit codes (comma-separated)
      --stop-when-contains string       stop when output contains pattern
      --stop-when-not-contains string   stop when output doesn't contain pattern
      --success-contains string         success if output contains pattern
      --success-on-exit string          consider these exit codes as success (comma-separated)
      --success-regex string            success if output matches regex
      --summary-only                    only show final summary
      --timeout string                  stop after duration (e.g., 5m, 30s)
  -v, --verbose                         enable verbose output
  -V, --verbose-output                  show detailed timing and condition info

Use "retry [command] --help" for more information about a command.
```

## Environment Variables

```bash
export RETRY_MAX_TRIES=5
export RETRY_BACKOFF=exponential  
export RETRY_BASE_DELAY=500ms
export RETRY_MAX_DELAY=30s
retry "your-command"
```


Demo:

![demo](doc/demo.gif)

## Using as a Go Library

In addition to the CLI tool, you can use `retry` as a Go library in your own applications.

### Installation

```bash
go get github.com/sgaunet/retry
```

### Basic Usage

```go
import (
    "context"
    "time"

    "github.com/sgaunet/retry/pkg/logger"
    "github.com/sgaunet/retry/pkg/retry"
)

func main() {
    // Create retry with max 5 attempts
    r, _ := retry.NewRetry("your-command", retry.NewStopOnMaxTries(5))

    // Add exponential backoff
    r.SetBackoffStrategy(retry.NewExponentialBackoff(
        time.Second,    // base delay
        time.Minute,    // max delay
        2.0,            // multiplier
    ))

    // Run with context and logger
    appLogger := logger.NewLogger("info")
    err := r.RunWithLogger(context.Background(), appLogger)
    if err != nil {
        // handle error
    }
}
```

### Composite Conditions

Combine multiple stop conditions with AND/OR logic:

```go
// Stop after 10 tries OR after 5 minutes
condition := retry.NewCompositeCondition(
    retry.LogicOR,
    retry.NewStopOnMaxTries(10),
    retry.NewStopOnTimeout(5 * time.Minute),
)
r, _ := retry.NewRetry("curl -sf https://api.example.com/health", condition)
```

### Success Conditions

Define custom success criteria beyond just exit code 0:

```go
r, _ := retry.NewRetry("curl https://api.example.com", retry.NewStopOnMaxTries(10))

// Consider successful if output contains "healthy"
successCond, _ := retry.NewSuccessContains("healthy")
r.SetSuccessConditions([]retry.ConditionRetryer{successCond})
```

### Available Backoff Strategies

- **Fixed**: Constant delay between retries
- **Exponential**: Delay doubles (or multiplies) each attempt
- **Linear**: Delay increases by a fixed increment
- **Fibonacci**: Delay follows the Fibonacci sequence
- **Jitter**: Wraps any strategy with randomness to avoid thundering herd
- **Custom**: User-defined delay sequence

### API Documentation

Full API documentation is available at: https://pkg.go.dev/github.com/sgaunet/retry/pkg/retry

See [examples/](examples/) for complete runnable examples.

## Real-World Examples

For comprehensive CLI usage examples, see the [examples/](examples/) directory:

- **CI/CD Integration**: [GitHub Actions](examples/ci-cd/github-actions.yml), [GitLab CI](examples/ci-cd/gitlab-ci.yml)
- **Docker & Containers**: [Health checks](examples/docker/healthcheck.sh), [Dockerfile](examples/docker/Dockerfile.example), [Compose](examples/docker/docker-compose.yml)
- **Databases**: [Migrations](examples/databases/migration-retry.sh), [Connection waiting](examples/databases/connection-wait.sh)
- **Networking**: [API health checks](examples/networking/api-endpoint-check.sh), [Downloads](examples/networking/download-with-retry.sh)
- **Kubernetes**: [Init containers](examples/kubernetes/init-container.yaml), [Jobs](examples/kubernetes/job-with-retry.yaml)
- **Testing**: [Flaky tests](examples/testing/flaky-tests.sh), [E2E tests](examples/testing/e2e-with-retry.sh)
- **Services**: [Startup waiting](examples/services/service-startup-wait.sh), [Dependencies](examples/services/dependency-check.sh)

## Shell Completion

`retry` ships completion scripts for bash, zsh, fish and PowerShell. They complete
subcommands, flags and flag values (backoff strategies, policy presets, log levels,
and the profiles declared in your config file).

### Bash

```bash
# Current session only
source <(retry completion bash)

# Persistent install (Linux)
retry completion bash > /etc/bash_completion.d/retry

# Persistent install (macOS, Homebrew)
retry completion bash > "$(brew --prefix)/etc/bash_completion.d/retry"
```

### Zsh

```bash
# Current session only
source <(retry completion zsh)

# Persistent install
retry completion zsh > "${fpath[1]}/_retry"
```

If completion is not enabled yet in your shell, add `autoload -U compinit; compinit`
to your `~/.zshrc` first.

### Fish

```bash
# Current session only
retry completion fish | source

# Persistent install
retry completion fish > ~/.config/fish/completions/retry.fish
```

### PowerShell

```powershell
# Current session only
retry completion powershell | Out-String | Invoke-Expression

# Persistent install
retry completion powershell >> $PROFILE
```

Once installed, tab completion suggests flags and their values:

```bash
retry --backoff <TAB>
# fixed  exponential  linear  fibonacci  custom

retry --policy <TAB>
# aggressive  cautious  database  fast  infinite  network  standard  test
```

# Install

## From binary 

Download the binary in the release section. 

## From Docker image

Docker registry is: ghcr.io/sgaunet/retry

The docker image is only interesting to copy the binary in your docker image.

# Development

This project is using :

* golang
* [task for development](https://taskfile.dev/#/)
* docker
* [docker buildx](https://github.com/docker/buildx)
* docker manifest
* [goreleaser](https://goreleaser.com/)
* [pre-commit](https://pre-commit.com/)

There are hooks executed in the precommit stage. Once the project cloned on your disk, please install pre-commit:

```
brew install pre-commit
```

Install tools:

```
task dev:install-prereq
```

And install the hooks:

```
task dev:install-pre-commit
```

If you like to launch manually the pre-commmit hook:

```
task dev:pre-commit
```
