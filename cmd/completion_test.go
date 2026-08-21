package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestFlagCompletionsRegistered verifies that every flag completion was wired
// successfully at init time (a failure means a flag name is misspelled).
func TestFlagCompletionsRegistered(t *testing.T) {
	if len(flagCompletionErrors) != 0 {
		t.Fatalf("flag completion registration failed: %v", errors.Join(flagCompletionErrors...))
	}

	expected := []string{
		"backoff", "log-level", "condition-logic", "config",
		"policy", "show-policy", "profile",
	}
	expected = append(expected, noFileCompletionFlags...)

	for _, name := range expected {
		if _, ok := rootCmd.GetFlagCompletionFunc(name); !ok {
			t.Errorf("no completion function registered for flag --%s", name)
		}
	}
}

// TestDefaultCompletionCmdDisabled verifies the explicit completion command
// replaces the one Cobra generates by default.
func TestDefaultCompletionCmdDisabled(t *testing.T) {
	if !rootCmd.CompletionOptions.DisableDefaultCmd {
		t.Error("Cobra default completion command should be disabled")
	}

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "completion" {
			found = true
			if cmd != completionCmd {
				t.Error("registered completion command is not the explicit one")
			}
		}
	}

	if !found {
		t.Fatal("completion command is not registered on the root command")
	}
}

// TestGenCompletionShells verifies a script is produced for every supported shell.
func TestGenCompletionShells(t *testing.T) {
	tests := []struct {
		shell    string
		contains string
	}{
		{shell: shellBash, contains: "bash completion V2 for retry"},
		{shell: shellZsh, contains: "#compdef retry"},
		{shell: shellFish, contains: "retry"},
		{shell: shellPowerShell, contains: "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			var out bytes.Buffer
			if err := genCompletion(rootCmd, &out, tt.shell); err != nil {
				t.Fatalf("genCompletion(%s) returned error: %v", tt.shell, err)
			}
			if out.Len() == 0 {
				t.Fatalf("genCompletion(%s) produced no output", tt.shell)
			}
			if !strings.Contains(out.String(), tt.contains) {
				t.Errorf("genCompletion(%s) output does not contain %q", tt.shell, tt.contains)
			}
		})
	}
}

// TestGenCompletionUnsupportedShell verifies unknown shells are rejected.
func TestGenCompletionUnsupportedShell(t *testing.T) {
	var out bytes.Buffer
	err := genCompletion(rootCmd, &out, "tcsh")

	if !errors.Is(err, ErrUnsupportedShell) {
		t.Fatalf("expected ErrUnsupportedShell, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output for unsupported shell, got %d bytes", out.Len())
	}
}

// TestCompletionCmdArgsValidation verifies the completion command accepts at
// most one argument, and only a supported shell name.
func TestCompletionCmdArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "valid shell", args: []string{shellBash}, wantErr: false},
		{name: "no argument shows help", args: []string{}, wantErr: false},
		{name: "two arguments", args: []string{shellBash, shellZsh}, wantErr: true},
		{name: "unknown shell", args: []string{"tcsh"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := completionCmd.Args(completionCmd, tt.args)
			if tt.wantErr && err == nil {
				t.Errorf("expected an error for args %v", tt.args)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for args %v: %v", tt.args, err)
			}
		})
	}
}

// TestCompletionCmdWithoutArgsShowsHelp verifies that running the command with
// no shell name prints usage instead of generating a script.
func TestCompletionCmdWithoutArgsShowsHelp(t *testing.T) {
	var out bytes.Buffer
	completionCmd.SetOut(&out)
	completionCmd.SetErr(&out)
	defer func() {
		completionCmd.SetOut(nil)
		completionCmd.SetErr(nil)
	}()

	if err := completionCmd.RunE(completionCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "To load completions") {
		t.Errorf("expected help output, got %q", out.String())
	}
	if strings.Contains(out.String(), "__retry_debug") {
		t.Error("expected help output, got a completion script")
	}
}

// TestCompletePolicyNames verifies policy presets are suggested with descriptions.
func TestCompletePolicyNames(t *testing.T) {
	completions, directive := completePolicyNames(rootCmd, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
	if len(completions) == 0 {
		t.Fatal("expected policy completions, got none")
	}

	joined := strings.Join(completions, "\n")
	for _, name := range []string{"fast", "standard", "network", "infinite"} {
		if !strings.Contains(joined, name) {
			t.Errorf("policy %q missing from completions", name)
		}
	}
	if !strings.Contains(completions[0], "\t") {
		t.Errorf("policy completion %q has no description", completions[0])
	}
}

// TestCompleteProfileNamesFromConfig verifies profiles are read from the config file.
func TestCompleteProfileNamesFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "retry.yaml")
	content := "profiles:\n  api-calls:\n    max_tries: 5\n  ci-tests:\n    max_tries: 2\n"

	const perms = 0o600
	if err := os.WriteFile(path, []byte(content), perms); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	previous := configFile
	configFile = path
	defer func() { configFile = previous }()

	completions, directive := completeProfileNames(rootCmd, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}

	want := []string{"api-calls", "ci-tests"}
	if len(completions) != len(want) {
		t.Fatalf("completions = %v, want %v", completions, want)
	}
	for i, name := range want {
		if completions[i] != name {
			t.Errorf("completions[%d] = %q, want %q (sorted order expected)", i, completions[i], name)
		}
	}
}

// TestCompleteProfileNamesWithoutConfig verifies that a missing config file
// yields no suggestions and no shell completion error.
func TestCompleteProfileNamesWithoutConfig(t *testing.T) {
	previous := configFile
	configFile = filepath.Join(t.TempDir(), "does-not-exist.yaml")
	defer func() { configFile = previous }()

	completions, directive := completeProfileNames(rootCmd, nil, "")

	if len(completions) != 0 {
		t.Errorf("expected no completions, got %v", completions)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}
