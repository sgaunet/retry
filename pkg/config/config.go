// Package config provides YAML configuration file support for the retry CLI.
// It handles loading, merging, and validating configuration profiles.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Configuration validation errors.
var (
	ErrProfileNotFound       = errors.New("profile not found in config file")
	ErrInvalidBackoff        = errors.New("invalid backoff strategy")
	ErrInvalidJitter         = errors.New("jitter must be between 0.0 and 1.0")
	ErrInvalidDuration       = errors.New("invalid duration")
	ErrInvalidLogLevel       = errors.New("invalid log level")
	ErrInvalidConditionLogic = errors.New("condition_logic must be 'and' or 'or'")
)

// Config represents the top-level configuration file structure.
type Config struct {
	Default  ProfileConfig            `yaml:"default"`
	Profiles map[string]ProfileConfig `yaml:"profiles"`
}

// ProfileConfig represents a single retry configuration profile.
// Pointer types are used for fields where zero is a meaningful value
// (e.g., max_tries: 0 means infinite retries).
type ProfileConfig struct {
	// Core retry settings
	MaxTries   *uint    `yaml:"max_tries,omitempty"`
	Delay      string   `yaml:"delay,omitempty"`
	Backoff    string   `yaml:"backoff,omitempty"`
	BaseDelay  string   `yaml:"base_delay,omitempty"`
	MaxDelay   string   `yaml:"max_delay,omitempty"`
	Multiplier *float64 `yaml:"multiplier,omitempty"`
	Increment  string   `yaml:"increment,omitempty"`
	Jitter     *float64 `yaml:"jitter,omitempty"`
	Delays     string   `yaml:"delays,omitempty"`

	// Stop conditions
	Timeout             string `yaml:"timeout,omitempty"`
	StopOnExit          string `yaml:"stop_on_exit,omitempty"`
	StopWhenContains    string `yaml:"stop_when_contains,omitempty"`
	StopWhenNotContains string `yaml:"stop_when_not_contains,omitempty"`
	StopAt              string `yaml:"stop_at,omitempty"`
	ConditionLogic      string `yaml:"condition_logic,omitempty"`

	// Success/failure conditions
	RetryOnExit     string `yaml:"retry_on_exit,omitempty"`
	SuccessOnExit   string `yaml:"success_on_exit,omitempty"`
	RetryIfContains string `yaml:"retry_if_contains,omitempty"`
	SuccessContains string `yaml:"success_contains,omitempty"`
	FailIfContains  string `yaml:"fail_if_contains,omitempty"`
	SuccessRegex    string `yaml:"success_regex,omitempty"`
	RetryRegex      string `yaml:"retry_regex,omitempty"`

	// Output control
	Quiet    *bool  `yaml:"quiet,omitempty"`
	LogFile  string `yaml:"log_file,omitempty"`
	LogLevel string `yaml:"log_level,omitempty"`
	JSON     *bool  `yaml:"json,omitempty"`

	// Policy reference (names a built-in policy as base)
	Policy string `yaml:"policy,omitempty"`
}

// Load parses a YAML configuration file and returns the Config.
func Load(path string) (*Config, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) //nolint:gosec // Path comes from user flag or known search locations
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// FindConfigFile searches for a configuration file in standard locations.
// If explicit is non-empty, it is returned directly. Otherwise, the search
// order is: ./retry.yaml, ./retry.yml, $HOME/.config/retry/retry.yaml, $HOME/.retryrc.
func FindConfigFile(explicit string) string {
	if explicit != "" {
		return explicit
	}

	candidates := []string{
		"retry.yaml",
		"retry.yml",
	}

	home, err := os.UserHomeDir()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".config", "retry", "retry.yaml"),
			filepath.Join(home, ".retryrc"),
		)
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetProfile merges the default section with the named profile.
// If name is empty, only the default section is returned.
func (c *Config) GetProfile(name string) (ProfileConfig, error) {
	if name == "" {
		return c.Default, nil
	}

	profile, ok := c.Profiles[name]
	if !ok {
		return ProfileConfig{}, fmt.Errorf("%w: %s", ErrProfileNotFound, name)
	}

	return MergeProfiles(c.Default, profile), nil
}

// MergeProfiles merges two profiles. Non-nil/non-zero fields in overlay
// take precedence over base fields.
//
//nolint:cyclop,funlen // Many fields to check, but each is a simple conditional assignment
func MergeProfiles(base, overlay ProfileConfig) ProfileConfig {
	result := base

	if overlay.MaxTries != nil {
		result.MaxTries = overlay.MaxTries
	}
	if overlay.Delay != "" {
		result.Delay = overlay.Delay
	}
	if overlay.Backoff != "" {
		result.Backoff = overlay.Backoff
	}
	if overlay.BaseDelay != "" {
		result.BaseDelay = overlay.BaseDelay
	}
	if overlay.MaxDelay != "" {
		result.MaxDelay = overlay.MaxDelay
	}
	if overlay.Multiplier != nil {
		result.Multiplier = overlay.Multiplier
	}
	if overlay.Increment != "" {
		result.Increment = overlay.Increment
	}
	if overlay.Jitter != nil {
		result.Jitter = overlay.Jitter
	}
	if overlay.Delays != "" {
		result.Delays = overlay.Delays
	}
	if overlay.Timeout != "" {
		result.Timeout = overlay.Timeout
	}
	if overlay.StopOnExit != "" {
		result.StopOnExit = overlay.StopOnExit
	}
	if overlay.StopWhenContains != "" {
		result.StopWhenContains = overlay.StopWhenContains
	}
	if overlay.StopWhenNotContains != "" {
		result.StopWhenNotContains = overlay.StopWhenNotContains
	}
	if overlay.StopAt != "" {
		result.StopAt = overlay.StopAt
	}
	if overlay.ConditionLogic != "" {
		result.ConditionLogic = overlay.ConditionLogic
	}
	if overlay.RetryOnExit != "" {
		result.RetryOnExit = overlay.RetryOnExit
	}
	if overlay.SuccessOnExit != "" {
		result.SuccessOnExit = overlay.SuccessOnExit
	}
	if overlay.RetryIfContains != "" {
		result.RetryIfContains = overlay.RetryIfContains
	}
	if overlay.SuccessContains != "" {
		result.SuccessContains = overlay.SuccessContains
	}
	if overlay.FailIfContains != "" {
		result.FailIfContains = overlay.FailIfContains
	}
	if overlay.SuccessRegex != "" {
		result.SuccessRegex = overlay.SuccessRegex
	}
	if overlay.RetryRegex != "" {
		result.RetryRegex = overlay.RetryRegex
	}
	if overlay.Quiet != nil {
		result.Quiet = overlay.Quiet
	}
	if overlay.LogFile != "" {
		result.LogFile = overlay.LogFile
	}
	if overlay.LogLevel != "" {
		result.LogLevel = overlay.LogLevel
	}
	if overlay.JSON != nil {
		result.JSON = overlay.JSON
	}
	if overlay.Policy != "" {
		result.Policy = overlay.Policy
	}

	return result
}

// ExpandEnvVars expands environment variable references in all string fields.
func (p *ProfileConfig) ExpandEnvVars() {
	v := reflect.ValueOf(p).Elem()
	t := v.Type()

	for i := range t.NumField() {
		field := v.Field(i)
		if field.Kind() == reflect.String && field.String() != "" {
			field.SetString(os.ExpandEnv(field.String()))
		}
	}
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if err := validateProfile("default", c.Default); err != nil {
		return err
	}

	for name, profile := range c.Profiles {
		if err := validateProfile(name, profile); err != nil {
			return err
		}
	}

	return nil
}

//nolint:cyclop // Validates many independent fields sequentially
func validateProfile(name string, p ProfileConfig) error {
	if p.Backoff != "" {
		valid := []string{"fixed", "exponential", "exp", "linear", "fibonacci", "fib", "custom"}
		if !slices.Contains(valid, p.Backoff) {
			return fmt.Errorf("profile %q: %w: %s", name, ErrInvalidBackoff, p.Backoff)
		}
	}

	if p.Jitter != nil {
		if *p.Jitter < 0 || *p.Jitter > 1 {
			return fmt.Errorf("profile %q: %w: got %f", name, ErrInvalidJitter, *p.Jitter)
		}
	}

	durationFields := map[string]string{
		"delay":      p.Delay,
		"base_delay": p.BaseDelay,
		"max_delay":  p.MaxDelay,
		"increment":  p.Increment,
		"timeout":    p.Timeout,
	}
	for field, val := range durationFields {
		if val != "" {
			if _, err := time.ParseDuration(val); err != nil {
				return fmt.Errorf("profile %q: %w for %s %q: %w", name, ErrInvalidDuration, field, val, err)
			}
		}
	}

	if p.LogLevel != "" {
		validLevels := []string{"error", "warn", "warning", "info", "debug"}
		if !slices.Contains(validLevels, strings.ToLower(p.LogLevel)) {
			return fmt.Errorf("profile %q: %w: %s", name, ErrInvalidLogLevel, p.LogLevel)
		}
	}

	if p.ConditionLogic != "" {
		upper := strings.ToUpper(p.ConditionLogic)
		if upper != "AND" && upper != "OR" {
			return fmt.Errorf("profile %q: %w: got %q", name, ErrInvalidConditionLogic, p.ConditionLogic)
		}
	}

	return nil
}

// GenerateTemplate returns a commented YAML template for retry config init.
func GenerateTemplate() string {
	return `# retry configuration file
# Place this file as ./retry.yaml, ~/.config/retry/retry.yaml, or ~/.retryrc

# Default settings apply to all retry invocations (unless overridden)
default:
  # max_tries: 3
  # delay: "1s"
  # backoff: "fixed"
  # base_delay: "1s"
  # max_delay: "5m"
  # jitter: 0.0
  # quiet: false
  # log_level: "info"

# Named profiles for different use cases
profiles:
  # Example: API calls with exponential backoff
  # api-calls:
  #   max_tries: 5
  #   backoff: "exponential"
  #   base_delay: "2s"
  #   max_delay: "1m"
  #   jitter: 0.2
  #   policy: "network"  # Base on a built-in policy

  # Example: CI test retries
  # ci-tests:
  #   max_tries: 3
  #   delay: "5s"
  #   backoff: "fixed"
  #   quiet: true
`
}

// FormatEffective returns a human-readable representation of the effective
// configuration for a given profile name.
//
//nolint:cyclop,funlen // Many fields to display, but each is a simple conditional print
func FormatEffective(c *Config, profileName string) string {
	profile, err := c.GetProfile(profileName)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	var b strings.Builder

	if profileName != "" {
		fmt.Fprintf(&b, "Profile: %s (merged with defaults)\n", profileName)
	} else {
		fmt.Fprintf(&b, "Profile: (defaults only)\n")
	}
	fmt.Fprintln(&b, "---")

	if profile.Policy != "" {
		fmt.Fprintf(&b, "policy: %s\n", profile.Policy)
	}
	if profile.MaxTries != nil {
		fmt.Fprintf(&b, "max_tries: %d\n", *profile.MaxTries)
	}
	if profile.Delay != "" {
		fmt.Fprintf(&b, "delay: %s\n", profile.Delay)
	}
	if profile.Backoff != "" {
		fmt.Fprintf(&b, "backoff: %s\n", profile.Backoff)
	}
	if profile.BaseDelay != "" {
		fmt.Fprintf(&b, "base_delay: %s\n", profile.BaseDelay)
	}
	if profile.MaxDelay != "" {
		fmt.Fprintf(&b, "max_delay: %s\n", profile.MaxDelay)
	}
	if profile.Multiplier != nil {
		fmt.Fprintf(&b, "multiplier: %.1f\n", *profile.Multiplier)
	}
	if profile.Increment != "" {
		fmt.Fprintf(&b, "increment: %s\n", profile.Increment)
	}
	if profile.Jitter != nil {
		fmt.Fprintf(&b, "jitter: %.2f\n", *profile.Jitter)
	}
	if profile.Delays != "" {
		fmt.Fprintf(&b, "delays: %s\n", profile.Delays)
	}
	if profile.Timeout != "" {
		fmt.Fprintf(&b, "timeout: %s\n", profile.Timeout)
	}
	if profile.StopOnExit != "" {
		fmt.Fprintf(&b, "stop_on_exit: %s\n", profile.StopOnExit)
	}
	if profile.StopWhenContains != "" {
		fmt.Fprintf(&b, "stop_when_contains: %s\n", profile.StopWhenContains)
	}
	if profile.StopWhenNotContains != "" {
		fmt.Fprintf(&b, "stop_when_not_contains: %s\n", profile.StopWhenNotContains)
	}
	if profile.StopAt != "" {
		fmt.Fprintf(&b, "stop_at: %s\n", profile.StopAt)
	}
	if profile.ConditionLogic != "" {
		fmt.Fprintf(&b, "condition_logic: %s\n", profile.ConditionLogic)
	}
	if profile.Quiet != nil {
		fmt.Fprintf(&b, "quiet: %t\n", *profile.Quiet)
	}
	if profile.LogFile != "" {
		fmt.Fprintf(&b, "log_file: %s\n", profile.LogFile)
	}
	if profile.LogLevel != "" {
		fmt.Fprintf(&b, "log_level: %s\n", profile.LogLevel)
	}
	if profile.JSON != nil {
		fmt.Fprintf(&b, "json: %t\n", *profile.JSON)
	}

	return b.String()
}
