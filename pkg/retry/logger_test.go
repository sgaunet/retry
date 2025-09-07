package retry

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLoggerJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		mode     OutputMode
		level    LogLevel
		wantJSON bool
	}{
		{
			name:     "JSON mode",
			mode:     OutputModeJSON,
			level:    LogLevelInfo,
			wantJSON: true,
		},
		{
			name:     "Normal mode",
			mode:     OutputModeNormal,
			level:    LogLevelInfo,
			wantJSON: false,
		},
		{
			name:     "Summary only mode",
			mode:     OutputModeSummaryOnly,
			level:    LogLevelQuiet,
			wantJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.level, tt.mode, true) // no color for testing
			logger.out = &buf

			// Simulate a retry execution
			logger.StartExecution("echo test", 2, "fixed")
			logger.StartAttempt(1)
			logger.LogCommandOutput("test output", false)
			logger.EndAttempt(0, true)
			logger.EndExecution(true, "", "")

			output := buf.String()

			if tt.wantJSON {
				// Validate JSON output
				var jsonOutput JSONOutput
				if err := json.Unmarshal([]byte(output), &jsonOutput); err != nil {
					t.Errorf("Failed to parse JSON output: %v\nOutput: %s", err, output)
				}

				// Validate JSON structure
				if jsonOutput.Command != "echo test" {
					t.Errorf("Expected command 'echo test', got '%s'", jsonOutput.Command)
				}
				if jsonOutput.TotalAttempts != 1 {
					t.Errorf("Expected 1 total attempt, got %d", jsonOutput.TotalAttempts)
				}
				if !jsonOutput.Successful {
					t.Errorf("Expected successful=true, got %v", jsonOutput.Successful)
				}
				if len(jsonOutput.Attempts) != 1 {
					t.Errorf("Expected 1 attempt in JSON, got %d", len(jsonOutput.Attempts))
				}
				if len(jsonOutput.Attempts) > 0 {
					attempt := jsonOutput.Attempts[0]
					if attempt.Output != "test output" {
						t.Errorf("Expected output 'test output', got '%s'", attempt.Output)
					}
					if attempt.ExitCode != 0 {
						t.Errorf("Expected exit code 0, got %d", attempt.ExitCode)
					}
					if !attempt.Success {
						t.Errorf("Expected attempt success=true, got %v", attempt.Success)
					}
				}
			} else {
				// For non-JSON modes, just ensure we got some output and it's not JSON
				if len(output) == 0 && tt.mode != OutputModeSummaryOnly {
					t.Errorf("Expected some output for mode %v, got none", tt.mode)
				}
				// Try to parse as JSON - it should fail for non-JSON modes
				var jsonOutput JSONOutput
				if err := json.Unmarshal([]byte(output), &jsonOutput); err == nil {
					t.Errorf("Output appears to be JSON when it shouldn't be. Output: %s", output)
				}
			}
		})
	}
}

func TestLoggerWithFile(t *testing.T) {
	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "retry_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close() // Close it so logger can create it

	logger := NewLoggerWithFile(LogLevelInfo, OutputModeNormal, true, tmpFile.Name())
	defer logger.Close()

	// Simulate logging
	logger.StartExecution("test command", 1, "fixed")
	logger.StartAttempt(1)
	logger.LogCommandOutput("test output", false)
	logger.EndAttempt(0, true)
	logger.EndExecution(true, "", "")

	// Read the log file
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	
	// Check that log file contains expected content
	expectedStrings := []string{
		"[1/1] Attempting command...",
		"[STDOUT] test output",
		"✓ Success",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(logContent, expected) {
			t.Errorf("Log file missing expected content: '%s'\nLog content: %s", expected, logContent)
		}
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
	}{
		{"Error", LogLevelError},
		{"Warn", LogLevelWarn}, 
		{"Info", LogLevelInfo},
		{"Debug", LogLevelDebug},
		{"Quiet", LogLevelQuiet},
		{"Normal", LogLevelNormal},
		{"Verbose", LogLevelVerbose},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.level, OutputModeNormal, true)
			logger.out = &buf
			logger.err = &buf

			// Test different log methods
			logger.Debug("debug message")
			logger.Info("info message") 
			logger.Warn("warn message")
			logger.Error("error message")
			logger.Verbose("verbose message")

			output := buf.String()
			
			// Different levels should show different amounts of output
			switch tt.level {
			case LogLevelError:
				if !strings.Contains(output, "ERROR: error message") {
					t.Errorf("Error level should show error messages")
				}
				if strings.Contains(output, "WARN:") || strings.Contains(output, "info message") {
					t.Errorf("Error level should not show warn/info messages")
				}
			case LogLevelWarn:
				if !strings.Contains(output, "ERROR: error message") || !strings.Contains(output, "WARN: warn message") {
					t.Errorf("Warn level should show error and warn messages")
				}
				if strings.Contains(output, "info message") {
					t.Errorf("Warn level should not show info messages")
				}
			case LogLevelInfo:
				if !strings.Contains(output, "ERROR: error message") || !strings.Contains(output, "WARN: warn message") || !strings.Contains(output, "info message") {
					t.Errorf("Info level should show error, warn, and info messages")
				}
				if strings.Contains(output, "DEBUG:") {
					t.Errorf("Info level should not show debug messages")
				}
			case LogLevelDebug:
				if !strings.Contains(output, "DEBUG: debug message") {
					t.Errorf("Debug level should show debug messages")
				}
			}
		})
	}
}

func TestJSONOutputStructure(t *testing.T) {
	logger := NewLogger(LogLevelInfo, OutputModeJSON, true)
	var buf bytes.Buffer
	logger.out = &buf

	// Simulate a multi-attempt retry with failure then success
	logger.StartExecution("flaky command", 3, "exponential")
	
	// First attempt - failure
	logger.StartAttempt(1)
	logger.LogCommandOutput("error: connection failed", true)
	logger.EndAttempt(1, false)
	
	// Second attempt - success
	logger.StartAttempt(2)
	logger.LogCommandOutput("success!", false)
	logger.EndAttempt(0, true)
	
	logger.EndExecution(true, "", "")

	// Parse JSON output
	var jsonOutput JSONOutput
	output := buf.String()
	if err := json.Unmarshal([]byte(output), &jsonOutput); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, output)
	}

	// Validate structure matches issue #21 requirements
	if jsonOutput.Command != "flaky command" {
		t.Errorf("Expected command 'flaky command', got '%s'", jsonOutput.Command)
	}
	if jsonOutput.TotalAttempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", jsonOutput.TotalAttempts)
	}
	if !jsonOutput.Successful {
		t.Errorf("Expected successful=true, got %v", jsonOutput.Successful)
	}
	if jsonOutput.MaxAttempts != 3 {
		t.Errorf("Expected max attempts=3, got %d", jsonOutput.MaxAttempts)
	}
	if jsonOutput.BackoffStrategy != "exponential" {
		t.Errorf("Expected backoff strategy 'exponential', got '%s'", jsonOutput.BackoffStrategy)
	}

	// Validate attempts array
	if len(jsonOutput.Attempts) != 2 {
		t.Fatalf("Expected 2 attempts in array, got %d", len(jsonOutput.Attempts))
	}

	// First attempt should be a failure
	attempt1 := jsonOutput.Attempts[0]
	if attempt1.Attempt != 1 {
		t.Errorf("Expected attempt number 1, got %d", attempt1.Attempt)
	}
	if attempt1.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", attempt1.ExitCode)
	}
	if attempt1.Success {
		t.Errorf("Expected first attempt success=false, got %v", attempt1.Success)
	}
	if !strings.Contains(attempt1.Output, "error: connection failed") {
		t.Errorf("Expected first attempt to contain error output, got '%s'", attempt1.Output)
	}

	// Second attempt should be success
	attempt2 := jsonOutput.Attempts[1]
	if attempt2.Attempt != 2 {
		t.Errorf("Expected attempt number 2, got %d", attempt2.Attempt)
	}
	if attempt2.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", attempt2.ExitCode)
	}
	if !attempt2.Success {
		t.Errorf("Expected second attempt success=true, got %v", attempt2.Success)
	}
	if !strings.Contains(attempt2.Output, "success!") {
		t.Errorf("Expected second attempt to contain success output, got '%s'", attempt2.Output)
	}

	// Validate timing fields exist
	if jsonOutput.StartTime.IsZero() {
		t.Errorf("Expected start time to be set")
	}
	if jsonOutput.EndTime.IsZero() {
		t.Errorf("Expected end time to be set")
	}
	if jsonOutput.TotalDuration == "" {
		t.Errorf("Expected total duration to be set")
	}

	for i, attempt := range jsonOutput.Attempts {
		if attempt.StartTime.IsZero() {
			t.Errorf("Expected start time for attempt %d to be set", i+1)
		}
		if attempt.EndTime.IsZero() {
			t.Errorf("Expected end time for attempt %d to be set", i+1)
		}
		if attempt.Duration == "" {
			t.Errorf("Expected duration for attempt %d to be set", i+1)
		}
	}
}