package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidYAML(t *testing.T) {
	content := `
default:
  max_tries: 5
  delay: "2s"
  backoff: "exponential"
profiles:
  api-calls:
    max_tries: 10
    base_delay: "1s"
    jitter: 0.2
`
	path := writeTemp(t, "retry.yaml", content)
	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, uint(5), *cfg.Default.MaxTries)
	assert.Equal(t, "2s", cfg.Default.Delay)
	assert.Equal(t, "exponential", cfg.Default.Backoff)

	api, ok := cfg.Profiles["api-calls"]
	require.True(t, ok)
	assert.Equal(t, uint(10), *api.MaxTries)
	assert.Equal(t, "1s", api.BaseDelay)
	assert.Equal(t, 0.2, *api.Jitter)
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := writeTemp(t, "bad.yaml", "not: [valid: yaml")
	_, err := Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoad_NonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/retry.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoad_ZeroMaxTries(t *testing.T) {
	content := `
default:
  max_tries: 0
`
	path := writeTemp(t, "zero.yaml", content)
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg.Default.MaxTries)
	assert.Equal(t, uint(0), *cfg.Default.MaxTries)
}

func TestFindConfigFile_Explicit(t *testing.T) {
	assert.Equal(t, "/my/config.yaml", FindConfigFile("/my/config.yaml"))
}

func TestFindConfigFile_LocalRetryYAML(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	require.NoError(t, os.WriteFile("retry.yaml", []byte("default:\n"), 0o644))
	assert.Equal(t, "retry.yaml", FindConfigFile(""))
}

func TestFindConfigFile_LocalRetryYML(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	require.NoError(t, os.WriteFile("retry.yml", []byte("default:\n"), 0o644))
	assert.Equal(t, "retry.yml", FindConfigFile(""))
}

func TestFindConfigFile_NoneFound(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	assert.Equal(t, "", FindConfigFile(""))
}

func TestGetProfile_DefaultOnly(t *testing.T) {
	five := uint(5)
	cfg := &Config{
		Default: ProfileConfig{MaxTries: &five, Delay: "1s"},
	}

	p, err := cfg.GetProfile("")
	require.NoError(t, err)
	assert.Equal(t, uint(5), *p.MaxTries)
	assert.Equal(t, "1s", p.Delay)
}

func TestGetProfile_MergesWithDefault(t *testing.T) {
	three := uint(3)
	ten := uint(10)
	cfg := &Config{
		Default: ProfileConfig{MaxTries: &three, Delay: "1s", Backoff: "fixed"},
		Profiles: map[string]ProfileConfig{
			"api": {MaxTries: &ten, Backoff: "exponential"},
		},
	}

	p, err := cfg.GetProfile("api")
	require.NoError(t, err)
	assert.Equal(t, uint(10), *p.MaxTries)
	assert.Equal(t, "exponential", p.Backoff)
	assert.Equal(t, "1s", p.Delay) // inherited from default
}

func TestGetProfile_Missing(t *testing.T) {
	cfg := &Config{}
	_, err := cfg.GetProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMergeProfiles_OverlayWins(t *testing.T) {
	five := uint(5)
	ten := uint(10)
	j1 := 0.1
	j2 := 0.3
	base := ProfileConfig{MaxTries: &five, Delay: "1s", Jitter: &j1, LogLevel: "info"}
	overlay := ProfileConfig{MaxTries: &ten, Jitter: &j2}

	result := MergeProfiles(base, overlay)
	assert.Equal(t, uint(10), *result.MaxTries)
	assert.Equal(t, "1s", result.Delay) // not overridden
	assert.Equal(t, 0.3, *result.Jitter)
	assert.Equal(t, "info", result.LogLevel) // not overridden
}

func TestMergeProfiles_NilPreservesBase(t *testing.T) {
	five := uint(5)
	base := ProfileConfig{MaxTries: &five}
	overlay := ProfileConfig{Delay: "2s"}

	result := MergeProfiles(base, overlay)
	assert.Equal(t, uint(5), *result.MaxTries)
	assert.Equal(t, "2s", result.Delay)
}

func TestMergeProfiles_ZeroPointerOverridesBase(t *testing.T) {
	five := uint(5)
	zero := uint(0)
	base := ProfileConfig{MaxTries: &five}
	overlay := ProfileConfig{MaxTries: &zero}

	result := MergeProfiles(base, overlay)
	require.NotNil(t, result.MaxTries)
	assert.Equal(t, uint(0), *result.MaxTries)
}

func TestExpandEnvVars(t *testing.T) {
	t.Setenv("TEST_DELAY", "5s")
	t.Setenv("TEST_LOG_FILE", "/var/log/retry.log")

	p := ProfileConfig{
		Delay:   "$TEST_DELAY",
		LogFile: "${TEST_LOG_FILE}",
		Backoff: "fixed", // no env ref, stays unchanged
	}
	p.ExpandEnvVars()

	assert.Equal(t, "5s", p.Delay)
	assert.Equal(t, "/var/log/retry.log", p.LogFile)
	assert.Equal(t, "fixed", p.Backoff)
}

func TestValidate_ValidConfig(t *testing.T) {
	five := uint(5)
	j := 0.2
	cfg := &Config{
		Default: ProfileConfig{
			MaxTries: &five,
			Delay:    "1s",
			Backoff:  "exponential",
			Jitter:   &j,
			LogLevel: "info",
		},
		Profiles: map[string]ProfileConfig{
			"test": {Delay: "500ms", Backoff: "fixed"},
		},
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidate_BadBackoff(t *testing.T) {
	cfg := &Config{
		Default: ProfileConfig{Backoff: "random"},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid backoff")
}

func TestValidate_BadJitter(t *testing.T) {
	j := 1.5
	cfg := &Config{
		Default: ProfileConfig{Jitter: &j},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jitter")
}

func TestValidate_NegativeJitter(t *testing.T) {
	j := -0.1
	cfg := &Config{
		Default: ProfileConfig{Jitter: &j},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jitter")
}

func TestValidate_BadDuration(t *testing.T) {
	cfg := &Config{
		Default: ProfileConfig{Delay: "not-a-duration"},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestValidate_BadLogLevel(t *testing.T) {
	cfg := &Config{
		Default: ProfileConfig{LogLevel: "verbose"},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidLogLevel)
}

func TestValidate_BadConditionLogic(t *testing.T) {
	cfg := &Config{
		Default: ProfileConfig{ConditionLogic: "xor"},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "condition_logic")
}

func TestValidate_ProfileError(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]ProfileConfig{
			"bad": {Backoff: "bogus"},
		},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `profile "bad"`)
}

func TestGenerateTemplate_ValidYAML(t *testing.T) {
	tmpl := GenerateTemplate()
	assert.Contains(t, tmpl, "default:")
	assert.Contains(t, tmpl, "profiles:")
}

func TestFormatEffective_Default(t *testing.T) {
	five := uint(5)
	cfg := &Config{
		Default: ProfileConfig{MaxTries: &five, Delay: "1s"},
	}
	out := FormatEffective(cfg, "")
	assert.Contains(t, out, "defaults only")
	assert.Contains(t, out, "max_tries: 5")
	assert.Contains(t, out, "delay: 1s")
}

func TestFormatEffective_NamedProfile(t *testing.T) {
	three := uint(3)
	ten := uint(10)
	cfg := &Config{
		Default: ProfileConfig{MaxTries: &three, Delay: "1s"},
		Profiles: map[string]ProfileConfig{
			"api": {MaxTries: &ten, Backoff: "exponential"},
		},
	}
	out := FormatEffective(cfg, "api")
	assert.Contains(t, out, "Profile: api")
	assert.Contains(t, out, "max_tries: 10")
	assert.Contains(t, out, "backoff: exponential")
	assert.Contains(t, out, "delay: 1s") // inherited
}

func TestFormatEffective_MissingProfile(t *testing.T) {
	cfg := &Config{}
	out := FormatEffective(cfg, "nope")
	assert.Contains(t, out, "Error")
}

func TestLoad_PolicyInProfile(t *testing.T) {
	content := `
profiles:
  net:
    policy: "network"
    max_tries: 3
`
	path := writeTemp(t, "policy.yaml", content)
	cfg, err := Load(path)
	require.NoError(t, err)

	p, ok := cfg.Profiles["net"]
	require.True(t, ok)
	assert.Equal(t, "network", p.Policy)
	assert.Equal(t, uint(3), *p.MaxTries)
}

func TestMergeProfiles_AllStringFields(t *testing.T) {
	base := ProfileConfig{}
	overlay := ProfileConfig{
		Timeout:             "5m",
		StopOnExit:          "0,1",
		StopWhenContains:    "ready",
		StopWhenNotContains: "error",
		StopAt:              "14:00",
		ConditionLogic:      "AND",
		RetryOnExit:         "1",
		SuccessOnExit:       "0",
		RetryIfContains:     "retry",
		SuccessContains:     "ok",
		FailIfContains:      "fatal",
		SuccessRegex:        "^OK",
		RetryRegex:          "^FAIL",
		LogFile:             "/tmp/log",
		LogLevel:            "debug",
		Policy:              "fast",
	}

	result := MergeProfiles(base, overlay)
	assert.Equal(t, "5m", result.Timeout)
	assert.Equal(t, "0,1", result.StopOnExit)
	assert.Equal(t, "ready", result.StopWhenContains)
	assert.Equal(t, "error", result.StopWhenNotContains)
	assert.Equal(t, "14:00", result.StopAt)
	assert.Equal(t, "AND", result.ConditionLogic)
	assert.Equal(t, "1", result.RetryOnExit)
	assert.Equal(t, "0", result.SuccessOnExit)
	assert.Equal(t, "retry", result.RetryIfContains)
	assert.Equal(t, "ok", result.SuccessContains)
	assert.Equal(t, "fatal", result.FailIfContains)
	assert.Equal(t, "^OK", result.SuccessRegex)
	assert.Equal(t, "^FAIL", result.RetryRegex)
	assert.Equal(t, "/tmp/log", result.LogFile)
	assert.Equal(t, "debug", result.LogLevel)
	assert.Equal(t, "fast", result.Policy)
}

func TestMergeProfiles_BoolPointers(t *testing.T) {
	trueVal := true
	falseVal := false

	base := ProfileConfig{Quiet: &trueVal}
	overlay := ProfileConfig{JSON: &falseVal}

	result := MergeProfiles(base, overlay)
	require.NotNil(t, result.Quiet)
	assert.True(t, *result.Quiet)
	require.NotNil(t, result.JSON)
	assert.False(t, *result.JSON)
}

func writeTemp(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
