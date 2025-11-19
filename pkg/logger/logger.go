// Package logger provides a simple logging interface for the retry package.
// It supports multiple log levels (Debug, Info, Warn, Error) and can be configured
// to use structured logging via slog or operate silently.
package logger

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	// logFilePermissions defines the file permissions for log files (rw-r--r--).
	logFilePermissions = 0644
)

var (
	// ErrEmptyFilepath is returned when an empty filepath is provided to NewFileLogger.
	ErrEmptyFilepath = errors.New("filepath cannot be empty")
)

// Logger defines the logging interface for retry operations.
// Implementations must support four log levels with variadic arguments
// for structured key-value pairs.
type Logger interface {
	// Debug logs a debug-level message with optional key-value pairs
	Debug(msg string, args ...any)
	// Info logs an info-level message with optional key-value pairs
	Info(msg string, args ...any)
	// Warn logs a warning-level message with optional key-value pairs
	Warn(msg string, args ...any)
	// Error logs an error-level message with optional key-value pairs
	Error(msg string, args ...any)
}

// slogLogger wraps slog.Logger to implement the Logger interface.
// It provides structured logging using the standard library's slog package.
type slogLogger struct {
	logger *slog.Logger
}

// Debug logs a debug-level message.
func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs an info-level message.
func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning-level message.
func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error-level message.
func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// noLogger is a silent logger implementation that discards all log messages.
// It's used when logging is disabled (e.g., quiet mode).
type noLogger struct{}

// Debug does nothing (silent logger).
//
//nolint:revive // Parameters required by Logger interface
func (n *noLogger) Debug(msg string, args ...any) {}

// Info does nothing (silent logger).
//
//nolint:revive // Parameters required by Logger interface
func (n *noLogger) Info(msg string, args ...any) {}

// Warn does nothing (silent logger).
//
//nolint:revive // Parameters required by Logger interface
func (n *noLogger) Warn(msg string, args ...any) {}

// Error does nothing (silent logger).
//
//nolint:revive // Parameters required by Logger interface
func (n *noLogger) Error(msg string, args ...any) {}

// parseLogLevel converts a string log level to slog.Level.
// Valid levels are: debug, info, warn, error (case-insensitive).
// Invalid levels default to info.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// NewLogger creates a new Logger instance with the specified log level string.
// Valid levels: debug, info, warn, error (case-insensitive).
// Invalid levels default to info. The logger writes to stdout.
//
//nolint:ireturn // Returning interface is intentional for dependency injection
func NewLogger(logLevel string) Logger {
	level := parseLogLevel(logLevel)
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &slogLogger{
		logger: slog.New(handler),
	}
}

// NewLoggerWithLevel creates a new Logger instance using slog with the specified level.
// The logger writes to stdout. This is useful when you already have an slog.Level value.
//
//nolint:ireturn // Returning interface is intentional for dependency injection
func NewLoggerWithLevel(level slog.Level) Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &slogLogger{
		logger: slog.New(handler),
	}
}

// NewNoLogger creates a silent logger that discards all log messages.
// Useful for quiet mode or when logging should be completely disabled.
//
//nolint:ireturn // Returning interface is intentional for dependency injection
func NewNoLogger() Logger {
	return &noLogger{}
}

// NoLogger returns a logger that suppresses all output.
// This is an alias for NewNoLogger() providing a more concise API.
// Useful for quiet mode or when logging should be completely disabled.
//
//nolint:ireturn // Returning interface is intentional for dependency injection
func NoLogger() Logger {
	return NewNoLogger()
}

// NewFileLogger creates a logger that writes to the specified file path.
// The file is created/truncated on each invocation with permissions 0644.
// Valid log levels: debug, info, warn, error (case-insensitive).
// Invalid levels default to info.
//
// Returns an error if:
//   - filepath is empty (ErrEmptyFilepath)
//   - file cannot be created or opened
//
//nolint:ireturn // Returning interface is intentional for dependency injection
func NewFileLogger(logLevel string, filepath string) (Logger, error) {
	if filepath == "" {
		return nil, ErrEmptyFilepath
	}

	// Create/truncate file with read/write permissions for owner, read for group and others
	//nolint:gosec // G304: File path is intentionally from user input for log file creation
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, logFilePermissions)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	level := parseLogLevel(logLevel)
	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: level,
	})

	return &slogLogger{
		logger: slog.New(handler),
	}, nil
}
