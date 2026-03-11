package retry

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestGetPolicy(t *testing.T) {
	defer goleak.VerifyNone(t)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid fast", "fast", false},
		{"valid standard", "standard", false},
		{"valid network", "network", false},
		{"valid database", "database", false},
		{"valid aggressive", "aggressive", false},
		{"valid cautious", "cautious", false},
		{"valid test", "test", false},
		{"valid infinite", "infinite", false},
		{"case insensitive", "FAST", false},
		{"mixed case", "Standard", false},
		{"unknown policy", "nonexistent", true},
		{"empty name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := GetPolicy(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("GetPolicy() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("GetPolicy() unexpected error: %v", err)
				return
			}
			if p.Name == "" {
				t.Error("GetPolicy() returned policy with empty name")
			}
		})
	}
}

func TestPolicyNames(t *testing.T) {
	defer goleak.VerifyNone(t)

	names := PolicyNames()

	if len(names) != 8 {
		t.Errorf("PolicyNames() returned %d names, want 8", len(names))
	}

	// Verify sorted order
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("PolicyNames() not sorted: %s comes after %s", names[i], names[i-1])
		}
	}

	// Verify all expected names are present
	expected := []string{"aggressive", "cautious", "database", "fast", "infinite", "network", "standard", "test"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("PolicyNames()[%d] = %s, want %s", i, names[i], name)
		}
	}
}

func TestFormatPolicyTable(t *testing.T) {
	defer goleak.VerifyNone(t)

	table := FormatPolicyTable()

	// Should contain all policy names
	for _, name := range PolicyNames() {
		if !containsString(table, name) {
			t.Errorf("FormatPolicyTable() missing policy: %s", name)
		}
	}

	// Should contain header
	if !containsString(table, "NAME") {
		t.Error("FormatPolicyTable() missing NAME header")
	}
	if !containsString(table, "BACKOFF") {
		t.Error("FormatPolicyTable() missing BACKOFF header")
	}
}

func TestFormatPolicyDetail(t *testing.T) {
	defer goleak.VerifyNone(t)

	tests := []struct {
		name    string
		policy  string
		wantErr bool
		checks  []string
	}{
		{
			name:    "valid standard",
			policy:  "standard",
			wantErr: false,
			checks:  []string{"standard", "exponential", "5", "1s", "30s", "2.0x"},
		},
		{
			name:    "valid network with jitter",
			policy:  "network",
			wantErr: false,
			checks:  []string{"network", "exponential", "Jitter: 20%"},
		},
		{
			name:    "valid database with increment",
			policy:  "database",
			wantErr: false,
			checks:  []string{"database", "linear", "Increment: 2s"},
		},
		{
			name:    "valid infinite",
			policy:  "infinite",
			wantErr: false,
			checks:  []string{"infinite", "infinite"},
		},
		{
			name:    "unknown policy",
			policy:  "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail, err := FormatPolicyDetail(tt.policy)
			if tt.wantErr {
				if err == nil {
					t.Error("FormatPolicyDetail() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("FormatPolicyDetail() unexpected error: %v", err)
				return
			}
			for _, check := range tt.checks {
				if !containsString(detail, check) {
					t.Errorf("FormatPolicyDetail(%s) missing: %s\ngot: %s", tt.policy, check, detail)
				}
			}
		})
	}
}

func TestPolicyFieldValues(t *testing.T) {
	defer goleak.VerifyNone(t)

	tests := []struct {
		name       string
		maxTries   uint
		backoff    string
		delay      time.Duration
		baseDelay  time.Duration
		maxDelay   time.Duration
		multiplier float64
		increment  time.Duration
		jitter     float64
	}{
		{
			name:     "fast",
			maxTries: 3,
			backoff:  "fixed",
			delay:    500 * time.Millisecond,
		},
		{
			name:       "standard",
			maxTries:   5,
			backoff:    "exponential",
			baseDelay:  time.Second,
			maxDelay:   30 * time.Second,
			multiplier: 2.0,
		},
		{
			name:      "network",
			maxTries:  10,
			backoff:   "exponential",
			baseDelay: 2 * time.Second,
			maxDelay:  2 * time.Minute,
			jitter:    0.2,
		},
		{
			name:      "database",
			maxTries:  10,
			backoff:   "linear",
			baseDelay: 3 * time.Second,
			increment: 2 * time.Second,
		},
		{
			name:      "aggressive",
			maxTries:  20,
			backoff:   "fibonacci",
			baseDelay: time.Second,
			maxDelay:  5 * time.Minute,
		},
		{
			name:       "cautious",
			maxTries:   5,
			backoff:    "exponential",
			baseDelay:  5 * time.Second,
			maxDelay:   10 * time.Minute,
			multiplier: 2.0,
		},
		{
			name:     "test",
			maxTries: 3,
			backoff:  "fixed",
			delay:    time.Second,
		},
		{
			name:       "infinite",
			maxTries:   0,
			backoff:    "exponential",
			baseDelay:  5 * time.Second,
			maxDelay:   5 * time.Minute,
			multiplier: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := GetPolicy(tt.name)
			if err != nil {
				t.Fatalf("GetPolicy(%s) unexpected error: %v", tt.name, err)
			}

			if p.MaxTries != tt.maxTries {
				t.Errorf("MaxTries = %d, want %d", p.MaxTries, tt.maxTries)
			}
			if p.Backoff != tt.backoff {
				t.Errorf("Backoff = %s, want %s", p.Backoff, tt.backoff)
			}
			if p.Delay != tt.delay {
				t.Errorf("Delay = %v, want %v", p.Delay, tt.delay)
			}
			if p.BaseDelay != tt.baseDelay {
				t.Errorf("BaseDelay = %v, want %v", p.BaseDelay, tt.baseDelay)
			}
			if p.MaxDelay != tt.maxDelay {
				t.Errorf("MaxDelay = %v, want %v", p.MaxDelay, tt.maxDelay)
			}
			if p.Multiplier != tt.multiplier {
				t.Errorf("Multiplier = %f, want %f", p.Multiplier, tt.multiplier)
			}
			if p.Increment != tt.increment {
				t.Errorf("Increment = %v, want %v", p.Increment, tt.increment)
			}
			if p.Jitter != tt.jitter {
				t.Errorf("Jitter = %f, want %f", p.Jitter, tt.jitter)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
