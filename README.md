[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/retry)](https://goreportcard.com/report/github.com/sgaunet/retry)


# template-cli

retry command will execute X times a failed command until it's successful. Interesting for flakky tests for example or to wait after something.

# Getting started

Usage is quite simple :

```
$ ./retry -h
Usage of retry:
  -c string
        command to execute
  -h    print help
  -m uint
        max tries of execution of failed command (default 3)
  -s uint
        sleep time in seconds between each try
  -version
        print version
```

# Install

## From binary 

Download the binary in the release section. 

## From Docker image

Docker registry is: sgaunet/retry

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
