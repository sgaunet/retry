package main

import (
	"sort"

	"github.com/sgaunet/retry/pkg/config"
	"github.com/sgaunet/retry/pkg/retry"
	"github.com/spf13/cobra"
)

// backoffCompletions lists the backoff strategies accepted by --backoff.
var backoffCompletions = []cobra.Completion{
	cobra.CompletionWithDesc("fixed", "Constant delay between attempts"),
	cobra.CompletionWithDesc("exponential", "Delay grows by --multiplier each attempt"),
	cobra.CompletionWithDesc("linear", "Delay grows by --increment each attempt"),
	cobra.CompletionWithDesc("fibonacci", "Delay follows the Fibonacci sequence"),
	cobra.CompletionWithDesc("custom", "Explicit delay list given by --delays"),
}

// logLevelCompletions lists the levels accepted by --log-level.
var logLevelCompletions = []cobra.Completion{
	cobra.CompletionWithDesc("error", "Errors only"),
	cobra.CompletionWithDesc("warn", "Warnings and errors"),
	cobra.CompletionWithDesc("info", "Default verbosity"),
	cobra.CompletionWithDesc("debug", "Full diagnostic output"),
}

// conditionLogicCompletions lists the operators accepted by --condition-logic.
var conditionLogicCompletions = []cobra.Completion{
	cobra.CompletionWithDesc("AND", "Stop only when every condition matches"),
	cobra.CompletionWithDesc("OR", "Stop as soon as any condition matches"),
}

// configFileExtensions restricts --config file completion to YAML files.
var configFileExtensions = []cobra.Completion{"yaml", "yml"}

// noFileCompletionFlags holds flags taking durations, numbers, codes or patterns,
// for which suggesting file names is never useful.
var noFileCompletionFlags = []string{
	"max-tries", "delay", "base-delay", "max-delay", "multiplier", "increment",
	"jitter", "delays", "timeout", "stop-at", "stop-on-exit", "stop-when-contains",
	"stop-when-not-contains", "retry-on-exit", "success-on-exit",
	"retry-if-contains", "success-contains", "fail-if-contains", "success-regex",
	"retry-regex",
}

// registerFlagCompletions wires value completions for the root command flags.
// It must be called after the flags have been declared.
func registerFlagCompletions() {
	registerFlagCompletion("backoff",
		cobra.FixedCompletions(backoffCompletions, cobra.ShellCompDirectiveNoFileComp))
	registerFlagCompletion("log-level",
		cobra.FixedCompletions(logLevelCompletions, cobra.ShellCompDirectiveNoFileComp))
	registerFlagCompletion("condition-logic",
		cobra.FixedCompletions(conditionLogicCompletions, cobra.ShellCompDirectiveNoFileComp))
	registerFlagCompletion("config",
		cobra.FixedCompletions(configFileExtensions, cobra.ShellCompDirectiveFilterFileExt))

	registerFlagCompletion("policy", completePolicyNames)
	registerFlagCompletion("show-policy", completePolicyNames)
	registerFlagCompletion("profile", completeProfileNames)

	for _, name := range noFileCompletionFlags {
		registerFlagCompletion(name, cobra.NoFileCompletions)
	}
}

// registerFlagCompletion registers fn for flagName, recording any failure.
func registerFlagCompletion(flagName string, fn cobra.CompletionFunc) {
	if err := rootCmd.RegisterFlagCompletionFunc(flagName, fn); err != nil {
		flagCompletionErrors = append(flagCompletionErrors, err)
	}
}

// completePolicyNames suggests the built-in retry policy presets.
func completePolicyNames(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	names := retry.PolicyNames()
	completions := make([]cobra.Completion, 0, len(names))

	for _, name := range names {
		policy, err := retry.GetPolicy(name)
		if err != nil {
			continue
		}
		completions = append(completions, cobra.CompletionWithDesc(name, policy.Description))
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeProfileNames suggests the profiles declared in the effective config file.
func completeProfileNames(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	path := config.FindConfigFile(configFile)
	if path == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.Load(path)
	if err != nil {
		// Completion must stay silent: an unreadable config is reported when the
		// command actually runs, not while the user is typing.
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]cobra.Completion, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	return names, cobra.ShellCompDirectiveNoFileComp
}
