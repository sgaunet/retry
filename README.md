[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/retry)](https://goreportcard.com/report/github.com/sgaunet/retry)
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
$ ./retry --help
Usage:
  retry [flags] "command"

Flags:
  -B, --backoff string      backoff strategy (fixed, exponential) (default "fixed")
  -b, --base-delay string   base delay for exponential backoff (default "1s")
  -d, --delay string        delay between retries (e.g., 1s, 500ms, 2m) (default "0s")
  -h, --help                help for retry
  -M, --max-delay string    maximum delay for exponential backoff (default "5m")
  -t, --max-tries uint      maximum number of retry attempts (0 for infinite) (default 3)
      --multiplier float    multiplier for exponential backoff (default 2)
  -v, --verbose             enable verbose output
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
