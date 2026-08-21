package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// ErrUnsupportedShell is returned when completion is requested for an unknown shell.
var ErrUnsupportedShell = errors.New("unsupported shell (supported: bash, zsh, fish, powershell)")

// Shells supported by the completion command.
const (
	shellBash       = "bash"
	shellZsh        = "zsh"
	shellFish       = "fish"
	shellPowerShell = "powershell"
)

// flagCompletionErrors collects flag completion registration failures.
// A non-empty slice means a flag name is wrong, which the unit tests assert against.
var flagCompletionErrors []error

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `Generate a shell completion script for retry.

To load completions:

Bash:
  $ source <(retry completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ retry completion bash > /etc/bash_completion.d/retry
  # macOS:
  $ retry completion bash > $(brew --prefix)/etc/bash_completion.d/retry

Zsh:
  $ source <(retry completion zsh)
  # To load completions for each session, execute once:
  $ retry completion zsh > "${fpath[1]}/_retry"

Fish:
  $ retry completion fish | source
  # To load completions for each session, execute once:
  $ retry completion fish > ~/.config/fish/completions/retry.fish

PowerShell:
  PS> retry completion powershell | Out-String | Invoke-Expression
  # To load completions for every session, add the output to your profile:
  PS> retry completion powershell >> $PROFILE
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []cobra.Completion{shellBash, shellZsh, shellFish, shellPowerShell},
	Args:                  cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Without a shell name, print the installation instructions instead of
		// failing, which is what the Cobra default command used to do.
		if len(args) == 0 {
			if err := cmd.Help(); err != nil {
				return fmt.Errorf("failed to display completion help: %w", err)
			}
			return nil
		}

		return genCompletion(cmd.Root(), cmd.OutOrStdout(), args[0])
	},
}

// genCompletion writes the completion script of the requested shell to out.
func genCompletion(root *cobra.Command, out io.Writer, shell string) error {
	var err error

	switch shell {
	case shellBash:
		// V2 is required for dynamic flag value completions (see registerFlagCompletions).
		err = root.GenBashCompletionV2(out, true)
	case shellZsh:
		err = root.GenZshCompletion(out)
	case shellFish:
		err = root.GenFishCompletion(out, true)
	case shellPowerShell:
		err = root.GenPowerShellCompletionWithDesc(out)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedShell, shell)
	}

	if err != nil {
		return fmt.Errorf("failed to generate %s completion script: %w", shell, err)
	}

	return nil
}

// setupCompletionCommand replaces the Cobra default completion command with
// the explicit one above and registers completions for flag values.
func setupCompletionCommand() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(completionCmd)

	registerFlagCompletions()
}
