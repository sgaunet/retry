// Package main implements a command line tool to retry a command
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"github.com/sgaunet/retry/pkg/retry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultMaxTries = 3
)

var (
	version = "dev"
	
	// Error definitions.
	ErrCommandRequired    = errors.New("command is required as a positional argument or via -c flag")
	ErrCommandEmpty       = errors.New("command is required")
	ErrDelayTooLarge      = errors.New("delay time too large")
)

var (
	maxTries uint
	delay    string
	verbose  bool
	command  string
)

var rootCmd = &cobra.Command{
	Use:   "retry [flags] \"command\"",
	Short: "Execute failed commands repeatedly until successful or limit reached",
	Long: `retry is a CLI tool that executes commands repeatedly until they succeed 
or a specified limit is reached. This is useful for handling flaky tests, 
waiting for services to become available, or dealing with transient failures.

The command to retry should be provided as a positional argument and quoted 
if it contains spaces or special characters.`,
	Example: `  # Basic usage
  retry "make test"

  # With custom retry count and delay
  retry --max-tries 10 --delay 5s "curl -f https://api.example.com"

  # Using short flags
  retry -t 5 -d 2s "flaky-command"

  # Using environment variables
  export RETRY_MAX_TRIES=3
  export RETRY_DELAY=1s
  retry "command-that-might-fail"

  # Backward compatibility (still supported)
  retry -c "old-style-command" -m 3 -s 2`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Handle backward compatibility for --version flag
		if cmd.Flags().Changed("version") {
			return nil
		}

		// Check if command is provided as positional argument
		if len(args) > 0 {
			command = strings.Join(args, " ")
			return nil
		}

		// Check if command is provided via -c flag (backward compatibility)
		if cmd.Flags().Changed("command") {
			return nil
		}

		return ErrCommandRequired
	},
	RunE:          runRetry,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version)
	},
}

func init() {
	// Add version subcommand
	rootCmd.AddCommand(versionCmd)

	// Modern flags
	rootCmd.Flags().UintVarP(&maxTries, "max-tries", "t", defaultMaxTries,
		"maximum number of retry attempts (0 for infinite)")
	rootCmd.Flags().StringVarP(&delay, "delay", "d", "0s", "delay between retries (e.g., 1s, 500ms, 2m)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Backward compatibility flags (hidden)
	rootCmd.Flags().UintP("m", "m", defaultMaxTries, "max tries (deprecated, use --max-tries)")
	rootCmd.Flags().UintP("s", "s", 0, "sleep time in seconds (deprecated, use --delay)")
	rootCmd.Flags().StringP("command", "c", "", "command to execute (deprecated, use positional argument)")
	rootCmd.Flags().Bool("version", false, "print version (deprecated, use version subcommand)")

	// Hide deprecated flags
	_ = rootCmd.Flags().MarkHidden("m")
	_ = rootCmd.Flags().MarkHidden("s")
	_ = rootCmd.Flags().MarkHidden("command")
	_ = rootCmd.Flags().MarkHidden("version")

	// Bind environment variables
	viper.SetEnvPrefix("RETRY")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind flags to viper
	_ = viper.BindPFlag("max-tries", rootCmd.Flags().Lookup("max-tries"))
	_ = viper.BindPFlag("delay", rootCmd.Flags().Lookup("delay"))
	_ = viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
}

func runRetry(cmd *cobra.Command, _ []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Handle backward compatibility
	if cmd.Flags().Changed("version") {
		fmt.Println(version)
		return nil
	}

	// Get command
	commandStr, err := getCommand(cmd)
	if err != nil {
		return err
	}

	// Parse configuration
	finalMaxTries := parseMaxTries(cmd)
	sleepDuration, err := parseDelay(cmd)
	if err != nil {
		return err
	}

	// Create and run retry
	return createAndRunRetry(commandStr, finalMaxTries, sleepDuration, logger)
}

func getCommand(cmd *cobra.Command) (string, error) {
	// Get command from -c flag if positional argument not provided
	if command == "" {
		if c, _ := cmd.Flags().GetString("command"); c != "" {
			command = c
		}
	}

	if command == "" {
		return "", ErrCommandEmpty
	}

	return command, nil
}

func parseMaxTries(cmd *cobra.Command) uint {
	finalMaxTries := maxTries

	// Handle backward compatibility for max tries
	if cmd.Flags().Changed("m") {
		if m, _ := cmd.Flags().GetUint("m"); m != 0 {
			finalMaxTries = m
		}
	}

	// Use environment variable if flag not explicitly set
	if !cmd.Flags().Changed("max-tries") && !cmd.Flags().Changed("m") {
		if envMaxTries := viper.GetUint("max-tries"); envMaxTries != 0 {
			finalMaxTries = envMaxTries
		}
	}

	return finalMaxTries
}

func parseDelay(cmd *cobra.Command) (time.Duration, error) {
	finalDelay := delay

	// Handle backward compatibility for sleep time
	if cmd.Flags().Changed("s") {
		if s, _ := cmd.Flags().GetUint("s"); s != 0 {
			finalDelay = fmt.Sprintf("%ds", s)
		}
	}

	// Use environment variable if flag not explicitly set
	if !cmd.Flags().Changed("delay") && !cmd.Flags().Changed("s") {
		if envDelay := viper.GetString("delay"); envDelay != "" {
			finalDelay = envDelay
		}
	}

	// Parse delay duration
	var sleepDuration time.Duration
	if finalDelay != "0s" && finalDelay != "" {
		var err error
		sleepDuration, err = time.ParseDuration(finalDelay)
		if err != nil {
			return 0, fmt.Errorf("invalid delay format: %w", err)
		}
	}

	return sleepDuration, nil
}

func createAndRunRetry(commandStr string, finalMaxTries uint, sleepDuration time.Duration, logger *slog.Logger) error {
	// Create retry instance
	r, err := retry.NewRetry(commandStr, retry.NewStopOnMaxTries(finalMaxTries))
	if err != nil {
		return fmt.Errorf("failed to create retry instance: %w", err)
	}

	// Set sleep function
	if sleepDuration > 0 {
		if sleepDuration > time.Duration(math.MaxInt64) {
			return ErrDelayTooLarge
		}

		r.SetSleep(func() { time.Sleep(sleepDuration) })
	}

	// Run the retry
	err = r.Run(logger)
	if err != nil {
		return fmt.Errorf("retry failed: %w", err)
	}

	return nil
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		// Show usage for command syntax errors, but not for retry failures
		if errors.Is(err, ErrCommandRequired) || errors.Is(err, ErrCommandEmpty) {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, "")
			_ = rootCmd.Usage()
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}