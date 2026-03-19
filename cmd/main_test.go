package main

import (
	"context"
	"errors"
	"testing"
)

// TestDetermineStatus verifies Bug 6:
// determineStatus must map errors to the correct JSON status string.
func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error returns success",
			err:      nil,
			expected: "success",
		},
		{
			name:     "context.Canceled returns interrupted",
			err:      context.Canceled,
			expected: "interrupted",
		},
		{
			name:     "context.DeadlineExceeded returns timeout",
			err:      context.DeadlineExceeded,
			expected: "timeout",
		},
		{
			name:     "wrapped context.Canceled returns interrupted",
			err:      errors.Join(errors.New("outer"), context.Canceled),
			expected: "interrupted",
		},
		{
			name:     "wrapped context.DeadlineExceeded returns timeout",
			err:      errors.Join(errors.New("outer"), context.DeadlineExceeded),
			expected: "timeout",
		},
		{
			name:     "arbitrary error returns failure",
			err:      errors.New("something went wrong"),
			expected: "failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineStatus(tt.err)
			if got != tt.expected {
				t.Errorf("determineStatus(%v) = %q, want %q", tt.err, got, tt.expected)
			}
		})
	}
}
