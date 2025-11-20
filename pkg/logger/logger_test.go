package logger

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoggerInterface verifies that our implementations satisfy the Logger interface.
func TestLoggerInterface(t *testing.T) {
	t.Run("slogLogger implements Logger", func(t *testing.T) {
		var _ Logger = &slogLogger{}
	})

	t.Run("noLogger implements Logger", func(t *testing.T) {
		var _ Logger = &noLogger{}
	})
}

// TestNewLogger verifies that NewLogger creates a working logger from string level.
func TestNewLogger(t *testing.T) {
	logger := NewLogger("info")
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	// Verify it's the correct type
	if _, ok := logger.(*slogLogger); !ok {
		t.Errorf("NewLogger returned wrong type: %T", logger)
	}

	// Verify methods don't panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

// TestNewLoggerWithLevel verifies that NewLoggerWithLevel creates a working logger.
func TestNewLoggerWithLevel(t *testing.T) {
	logger := NewLoggerWithLevel(slog.LevelInfo)
	if logger == nil {
		t.Fatal("NewLoggerWithLevel returned nil")
	}

	// Verify it's the correct type
	if _, ok := logger.(*slogLogger); !ok {
		t.Errorf("NewLoggerWithLevel returned wrong type: %T", logger)
	}

	// Verify methods don't panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

// TestNewNoLogger verifies that NewNoLogger creates a silent logger.
func TestNewNoLogger(t *testing.T) {
	logger := NewNoLogger()
	if logger == nil {
		t.Fatal("NewNoLogger returned nil")
	}

	// Verify it's the correct type
	if _, ok := logger.(*noLogger); !ok {
		t.Errorf("NewNoLogger returned wrong type: %T", logger)
	}

	// Verify methods don't panic (they should do nothing)
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

// TestNoLogger verifies that NoLogger creates a silent logger.
func TestNoLogger(t *testing.T) {
	logger := NoLogger()
	if logger == nil {
		t.Fatal("NoLogger returned nil")
	}

	// Verify it's the correct type
	if _, ok := logger.(*noLogger); !ok {
		t.Errorf("NoLogger returned wrong type: %T", logger)
	}

	// Verify methods don't panic (they should do nothing)
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

// TestNoLoggerEquivalence verifies that NoLogger and NewNoLogger produce equivalent loggers.
func TestNoLoggerEquivalence(t *testing.T) {
	logger1 := NoLogger()
	logger2 := NewNoLogger()

	// Both should be noLogger instances
	if _, ok := logger1.(*noLogger); !ok {
		t.Errorf("NoLogger() returned wrong type: %T", logger1)
	}
	if _, ok := logger2.(*noLogger); !ok {
		t.Errorf("NewNoLogger() returned wrong type: %T", logger2)
	}

	// Verify both work without panicking
	logger1.Info("test from NoLogger")
	logger2.Info("test from NewNoLogger")
}

// TestParseLogLevel verifies that parseLogLevel correctly converts strings to slog.Level.
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{"debug lowercase", "debug", slog.LevelDebug},
		{"info lowercase", "info", slog.LevelInfo},
		{"warn lowercase", "warn", slog.LevelWarn},
		{"error lowercase", "error", slog.LevelError},
		{"DEBUG uppercase", "DEBUG", slog.LevelDebug},
		{"INFO uppercase", "INFO", slog.LevelInfo},
		{"WARN uppercase", "WARN", slog.LevelWarn},
		{"ERROR uppercase", "ERROR", slog.LevelError},
		{"Debug mixed case", "Debug", slog.LevelDebug},
		{"WaRn mixed case", "WaRn", slog.LevelWarn},
		{"invalid defaults to info", "invalid", slog.LevelInfo},
		{"empty defaults to info", "", slog.LevelInfo},
		{"random defaults to info", "xyz123", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNewLoggerLevels verifies that NewLogger correctly handles different log level strings.
func TestNewLoggerLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
		{"DEBUG uppercase", "DEBUG"},
		{"invalid defaults to info", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level)
			if logger == nil {
				t.Fatalf("NewLogger(%q) returned nil", tt.level)
			}
			if _, ok := logger.(*slogLogger); !ok {
				t.Errorf("NewLogger(%q) returned wrong type: %T", tt.level, logger)
			}
		})
	}
}

// TestSlogLoggerMethods verifies that slogLogger methods work correctly.
func TestSlogLoggerMethods(t *testing.T) {
	logger := NewLogger("debug")
	slogImpl, ok := logger.(*slogLogger)
	if !ok {
		t.Fatalf("expected *slogLogger, got %T", logger)
	}

	// Test all methods with various argument patterns
	tests := []struct {
		name string
		fn   func(string, ...any)
	}{
		{"Debug", slogImpl.Debug},
		{"Info", slogImpl.Info},
		{"Warn", slogImpl.Warn},
		{"Error", slogImpl.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with no args
			tt.fn("message without args")

			// Test with key-value pairs
			tt.fn("message with args", "key1", "value1", "key2", 42)

			// Test with odd number of args (slog handles this gracefully)
			tt.fn("message with odd args", "key1", "value1", "orphan")
		})
	}
}

// TestNoLoggerMethods verifies that noLogger methods are silent.
func TestNoLoggerMethods(t *testing.T) {
	logger := NewNoLogger()
	noImpl, ok := logger.(*noLogger)
	if !ok {
		t.Fatalf("expected *noLogger, got %T", logger)
	}

	// Test all methods - they should do nothing and not panic
	tests := []struct {
		name string
		fn   func(string, ...any)
	}{
		{"Debug", noImpl.Debug},
		{"Info", noImpl.Info},
		{"Warn", noImpl.Warn},
		{"Error", noImpl.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn("message", "key", "value")
		})
	}
}

// BenchmarkSlogLogger benchmarks the slog-based logger.
func BenchmarkSlogLogger(b *testing.B) {
	logger := NewLogger("info")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

// BenchmarkSlogLoggerWithLevel benchmarks the slog-based logger using NewLoggerWithLevel.
func BenchmarkSlogLoggerWithLevel(b *testing.B) {
	logger := NewLoggerWithLevel(slog.LevelInfo)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

// BenchmarkNoLogger benchmarks the silent logger.
func BenchmarkNoLogger(b *testing.B) {
	logger := NewNoLogger()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

// TestNewFileLogger verifies that NewFileLogger creates a logger that writes to a file.
func TestNewFileLogger(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger("info", logFile)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	if logger == nil {
		t.Fatal("NewFileLogger returned nil logger")
	}

	// Verify it's the correct type
	if _, ok := logger.(*slogLogger); !ok {
		t.Errorf("NewFileLogger returned wrong type: %T", logger)
	}

	// Write log messages
	logger.Debug("debug message")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message")
	logger.Error("error message")

	// Read file and verify content was written
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	// Debug should not appear (level is info)
	if strings.Contains(contentStr, "debug message") {
		t.Error("Debug message appeared in log file with info level")
	}
	// Info, Warn, Error should appear
	if !strings.Contains(contentStr, "info message") {
		t.Error("Info message not found in log file")
	}
	if !strings.Contains(contentStr, "warn message") {
		t.Error("Warn message not found in log file")
	}
	if !strings.Contains(contentStr, "error message") {
		t.Error("Error message not found in log file")
	}
}

// TestNewFileLoggerEmptyPath verifies that NewFileLogger returns error for empty filepath.
func TestNewFileLoggerEmptyPath(t *testing.T) {
	logger, err := NewFileLogger("info", "")
	if err == nil {
		t.Fatal("NewFileLogger with empty filepath should return error")
	}
	if logger != nil {
		t.Error("NewFileLogger with empty filepath should return nil logger")
	}
	if !errors.Is(err, ErrEmptyFilepath) {
		t.Errorf("Expected ErrEmptyFilepath, got: %v", err)
	}
}

// TestNewFileLoggerInvalidPath verifies error handling for invalid paths.
func TestNewFileLoggerInvalidPath(t *testing.T) {
	// Use a path that should fail on most systems (directory doesn't exist)
	invalidPath := "/nonexistent/directory/that/should/not/exist/test.log"
	logger, err := NewFileLogger("info", invalidPath)
	if err == nil {
		t.Fatal("NewFileLogger with invalid path should return error")
	}
	if logger != nil {
		t.Error("NewFileLogger with invalid path should return nil logger")
	}
	if !strings.Contains(err.Error(), "failed to open log file") {
		t.Errorf("Expected 'failed to open log file' error, got: %v", err)
	}
}

// TestNewFileLoggerTruncation verifies that file is truncated on each creation.
func TestNewFileLoggerTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "truncate.log")

	// Create first logger and write content
	logger1, err := NewFileLogger("info", logFile)
	if err != nil {
		t.Fatalf("First NewFileLogger failed: %v", err)
	}
	logger1.Info("first message")

	// Create second logger (should truncate)
	logger2, err := NewFileLogger("info", logFile)
	if err != nil {
		t.Fatalf("Second NewFileLogger failed: %v", err)
	}
	logger2.Info("second message")

	// Read file - should only contain second message
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "first message") {
		t.Error("File was not truncated - first message still present")
	}
	if !strings.Contains(contentStr, "second message") {
		t.Error("Second message not found in truncated file")
	}
}

// TestNewFileLoggerPermissions verifies file is created with correct permissions.
func TestNewFileLoggerPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "perms.log")

	_, err := NewFileLogger("info", logFile)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}

	// Check file permissions
	fileInfo, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	mode := fileInfo.Mode()
	expectedPerm := os.FileMode(0644)
	if mode.Perm() != expectedPerm {
		t.Errorf("File permissions = %o, want %o", mode.Perm(), expectedPerm)
	}
}

// TestNewFileLoggerLevels verifies different log levels work correctly with file logger.
func TestNewFileLoggerLevels(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		shouldContain []string
		shouldNotContain []string
	}{
		{
			name:          "debug level",
			level:         "debug",
			shouldContain: []string{"level=DEBUG", "level=INFO", "level=WARN", "level=ERROR"},
			shouldNotContain: []string{},
		},
		{
			name:          "info level",
			level:         "info",
			shouldContain: []string{"level=INFO", "level=WARN", "level=ERROR"},
			shouldNotContain: []string{"level=DEBUG"},
		},
		{
			name:          "warn level",
			level:         "warn",
			shouldContain: []string{"level=WARN", "level=ERROR"},
			shouldNotContain: []string{"level=DEBUG", "level=INFO"},
		},
		{
			name:          "error level",
			level:         "error",
			shouldContain: []string{"level=ERROR"},
			shouldNotContain: []string{"level=DEBUG", "level=INFO", "level=WARN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logFile := filepath.Join(tmpDir, "level_test.log")

			logger, err := NewFileLogger(tt.level, logFile)
			if err != nil {
				t.Fatalf("NewFileLogger failed: %v", err)
			}

			// Write all log levels
			logger.Debug("debug msg")
			logger.Info("info msg")
			logger.Warn("warn msg")
			logger.Error("error msg")

			// Read and verify
			content, err := os.ReadFile(logFile)
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}

			contentStr := string(content)
			for _, expected := range tt.shouldContain {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected to find %q in log file", expected)
				}
			}
			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(contentStr, unexpected) {
					t.Errorf("Did not expect to find %q in log file", unexpected)
				}
			}
		})
	}
}
