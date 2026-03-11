package retry

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrUnknownPolicy is returned when a policy name is not found in the registry.
var ErrUnknownPolicy = errors.New("unknown policy")

// Policy represents a named retry configuration preset that bundles
// best-practice retry settings into a single convenient name.
type Policy struct {
	Name        string
	Description string
	MaxTries    uint
	Backoff     string        // "fixed", "exponential", "linear", "fibonacci"
	Delay       time.Duration // For fixed backoff
	BaseDelay   time.Duration // For non-fixed strategies
	MaxDelay    time.Duration
	Multiplier  float64       // 0 = use default 2.0
	Increment   time.Duration // For linear backoff
	Jitter      float64       // 0.0-1.0
}

// policies is the registry of named retry policy presets.
//
//nolint:mnd // Policy presets are intentional configuration values, not magic numbers
var policies = map[string]Policy{
	"fast": {
		Name:        "fast",
		Description: "Quick retries for local/fast operations",
		MaxTries:    3,
		Backoff:     "fixed",
		Delay:       500 * time.Millisecond,
	},
	"standard": {
		Name:        "standard",
		Description: "Balanced retry with exponential backoff",
		MaxTries:    5,
		Backoff:     "exponential",
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	},
	"network": {
		Name:        "network",
		Description: "Network-aware retries with jitter to avoid thundering herd",
		MaxTries:    10,
		Backoff:     "exponential",
		BaseDelay:   2 * time.Second,
		MaxDelay:    2 * time.Minute,
		Jitter:      0.2,
	},
	"database": {
		Name:        "database",
		Description: "Linear backoff for database reconnection",
		MaxTries:    10,
		Backoff:     "linear",
		BaseDelay:   3 * time.Second,
		Increment:   2 * time.Second,
	},
	"aggressive": {
		Name:        "aggressive",
		Description: "Many retries with fibonacci backoff for stubborn services",
		MaxTries:    20,
		Backoff:     "fibonacci",
		BaseDelay:   time.Second,
		MaxDelay:    5 * time.Minute,
	},
	"cautious": {
		Name:        "cautious",
		Description: "Conservative retries with long delays",
		MaxTries:    5,
		Backoff:     "exponential",
		BaseDelay:   5 * time.Second,
		MaxDelay:    10 * time.Minute,
		Multiplier:  2.0,
	},
	"test": {
		Name:        "test",
		Description: "Simple preset for testing and CI",
		MaxTries:    3,
		Backoff:     "fixed",
		Delay:       time.Second,
	},
	"infinite": {
		Name:        "infinite",
		Description: "Unlimited retries with exponential backoff",
		MaxTries:    0,
		Backoff:     "exponential",
		BaseDelay:   5 * time.Second,
		MaxDelay:    5 * time.Minute,
		Multiplier:  2.0,
	},
}

// GetPolicy returns the policy with the given name.
func GetPolicy(name string) (Policy, error) {
	p, ok := policies[strings.ToLower(name)]
	if !ok {
		return Policy{}, fmt.Errorf("%w: %s", ErrUnknownPolicy, name)
	}
	return p, nil
}

// PolicyNames returns a sorted list of all available policy names.
func PolicyNames() []string {
	names := make([]string, 0, len(policies))
	for name := range policies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FormatPolicyTable returns a tabular summary of all policies for display.
func FormatPolicyTable() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-12s %-12s %-6s %-10s %s\n",
		"NAME", "BACKOFF", "TRIES", "DELAY", "DESCRIPTION")
	fmt.Fprintf(&b, "%-12s %-12s %-6s %-10s %s\n",
		"----", "-------", "-----", "-----", "-----------")

	for _, name := range PolicyNames() {
		p := policies[name]
		tries := strconv.FormatUint(uint64(p.MaxTries), 10)
		if p.MaxTries == 0 {
			tries = "inf"
		}
		delayStr := formatPolicyDelay(p)
		fmt.Fprintf(&b, "%-12s %-12s %-6s %-10s %s\n",
			p.Name, p.Backoff, tries, delayStr, p.Description)
	}

	return b.String()
}

// formatPolicyDelay returns the primary delay value for display.
func formatPolicyDelay(p Policy) string {
	if p.Delay > 0 {
		return p.Delay.String()
	}
	if p.BaseDelay > 0 {
		return p.BaseDelay.String()
	}
	return "-"
}

// FormatPolicyDetail returns a detailed description of a single policy.
func FormatPolicyDetail(name string) (string, error) {
	p, err := GetPolicy(name)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Policy: %s\n", p.Name)
	fmt.Fprintf(&b, "Description: %s\n", p.Description)
	fmt.Fprintf(&b, "Max Tries: %s\n", formatTries(p.MaxTries))
	fmt.Fprintf(&b, "Backoff: %s\n", p.Backoff)

	if p.Delay > 0 {
		fmt.Fprintf(&b, "Delay: %s\n", p.Delay)
	}
	if p.BaseDelay > 0 {
		fmt.Fprintf(&b, "Base Delay: %s\n", p.BaseDelay)
	}
	if p.MaxDelay > 0 {
		fmt.Fprintf(&b, "Max Delay: %s\n", p.MaxDelay)
	}
	if p.Multiplier > 0 {
		fmt.Fprintf(&b, "Multiplier: %.1fx\n", p.Multiplier)
	}
	if p.Increment > 0 {
		fmt.Fprintf(&b, "Increment: %s\n", p.Increment)
	}
	if p.Jitter > 0 {
		fmt.Fprintf(&b, "Jitter: %.0f%%\n", p.Jitter*100) //nolint:mnd // percentage conversion
	}

	return b.String(), nil
}

// formatTries formats the max tries value for display.
func formatTries(maxTries uint) string {
	if maxTries == 0 {
		return "infinite"
	}
	return strconv.FormatUint(uint64(maxTries), 10)
}
