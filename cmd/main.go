// Package main implements a command line tool to retry a command
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/sgaunet/retry/pkg/retry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultMaxTries    = 3
	defaultMultiplier  = 2.0
)

var (
	version = "dev"
	
	// Error definitions.
	ErrCommandRequired       = errors.New("command is required as a positional argument")
	ErrCommandEmpty          = errors.New("command is required")
	ErrInvalidMultiplier     = errors.New("multiplier must be greater than 1.0")
	ErrUnsupportedBackoff = errors.New(
		"unsupported backoff strategy (supported: fixed, exponential, linear, fibonacci, custom)")
	ErrInvalidJitter         = errors.New("jitter must be between 0.0 and 1.0")
	ErrEmptyDelays           = errors.New("delays cannot be empty when using custom backoff")
)

var (
	maxTries    uint
	delay       string
	verbose     bool
	backoff     string
	baseDelay   string
	maxDelay    string
	multiplier  float64
	increment   string
	jitter      float64
	delays      string
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

  # Exponential backoff examples
  retry --backoff exponential --base-delay 1s "flaky-command"
  
  # Linear backoff with increment
  retry --backoff linear --base-delay 1s --increment 500ms "command"
  
  # Fibonacci backoff
  retry --backoff fibonacci --base-delay 1s "command"
  
  # Custom delays
  retry --backoff custom --delays "1s,2s,5s,10s,30s" "command"
  
  # With jitter for preventing thundering herd
  retry --backoff exponential --jitter 0.2 "command"`,
	Args: func(_ *cobra.Command, args []string) error {
		// Check if command is provided as positional argument
		if len(args) > 0 {
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
	
	// Backoff strategy flags
	rootCmd.Flags().StringVarP(&backoff, "backoff", "B", "fixed",
		"backoff strategy (fixed, exponential, linear, fibonacci, custom)")
	rootCmd.Flags().StringVarP(&baseDelay, "base-delay", "b", "1s", "base delay for backoff strategies")
	rootCmd.Flags().StringVarP(&maxDelay, "max-delay", "M", "5m", "maximum delay cap for backoff strategies")
	rootCmd.Flags().Float64Var(&multiplier, "multiplier", defaultMultiplier, "multiplier for exponential backoff")
	rootCmd.Flags().StringVar(&increment, "increment", "500ms", "increment for linear backoff")
	rootCmd.Flags().Float64VarP(&jitter, "jitter", "j", 0.0, "jitter percentage (0.0-1.0) to add randomness")
	rootCmd.Flags().StringVar(&delays, "delays", "", "comma-separated custom delays (e.g., 1s,2s,5s,10s)")


	// Bind environment variables
	viper.SetEnvPrefix("RETRY")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind flags to viper
	_ = viper.BindPFlag("max-tries", rootCmd.Flags().Lookup("max-tries"))
	_ = viper.BindPFlag("delay", rootCmd.Flags().Lookup("delay"))
	_ = viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	_ = viper.BindPFlag("backoff", rootCmd.Flags().Lookup("backoff"))
	_ = viper.BindPFlag("base-delay", rootCmd.Flags().Lookup("base-delay"))
	_ = viper.BindPFlag("max-delay", rootCmd.Flags().Lookup("max-delay"))
	_ = viper.BindPFlag("multiplier", rootCmd.Flags().Lookup("multiplier"))
	_ = viper.BindPFlag("increment", rootCmd.Flags().Lookup("increment"))
	_ = viper.BindPFlag("jitter", rootCmd.Flags().Lookup("jitter"))
	_ = viper.BindPFlag("delays", rootCmd.Flags().Lookup("delays"))
}

func runRetry(cmd *cobra.Command, args []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Get command from positional arguments
	commandStr := strings.Join(args, " ")
	if commandStr == "" {
		return ErrCommandEmpty
	}

	// Parse configuration
	finalMaxTries := parseMaxTries(cmd)
	
	// Parse backoff strategy
	backoffType := getBackoffType(cmd)
	var backoffStrategy retry.BackoffStrategy
	var err error
	
	switch backoffType {
	case "fixed":
		backoffStrategy, err = parseFixedBackoff(cmd)
	case "exponential", "exp":
		backoffStrategy, err = parseExponentialBackoff(cmd)
	case "linear":
		backoffStrategy, err = parseLinearBackoff(cmd)
	case "fibonacci", "fib":
		backoffStrategy, err = parseFibonacciBackoff(cmd)
	case "custom":
		backoffStrategy, err = parseCustomBackoff(cmd)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedBackoff, backoffType)
	}
	
	if err != nil {
		return err
	}
	
	// Apply jitter if specified
	jitterWrapper, err := applyJitter(cmd, backoffStrategy)
	if err != nil {
		return err
	}
	if jitterWrapper != nil {
		backoffStrategy = jitterWrapper
	}

	// Create and run retry
	return createAndRunRetry(commandStr, finalMaxTries, backoffStrategy, logger)
}

func parseMaxTries(cmd *cobra.Command) uint {
	finalMaxTries := maxTries

	// Use environment variable if flag not explicitly set
	if !cmd.Flags().Changed("max-tries") {
		if envMaxTries := viper.GetUint("max-tries"); envMaxTries != 0 {
			finalMaxTries = envMaxTries
		}
	}

	return finalMaxTries
}

func parseDelay(cmd *cobra.Command) (time.Duration, error) {
	finalDelay := delay

	// Use environment variable if flag not explicitly set
	if !cmd.Flags().Changed("delay") {
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


func getBackoffType(cmd *cobra.Command) string {
	backoffType := backoff
	if !cmd.Flags().Changed("backoff") {
		if envBackoff := viper.GetString("backoff"); envBackoff != "" {
			backoffType = envBackoff
		}
	}
	return backoffType
}

func parseFixedBackoff(cmd *cobra.Command) (*retry.FixedBackoff, error) {
	sleepDuration, err := parseDelay(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse delay for fixed backoff: %w", err)
	}
	return retry.NewFixedBackoff(sleepDuration), nil
}

func parseExponentialBackoff(cmd *cobra.Command) (*retry.ExponentialBackoff, error) {
	baseDuration, err := parseBaseDuration(cmd)
	if err != nil {
		return nil, err
	}

	maxDuration, err := parseMaxDuration(cmd)
	if err != nil {
		return nil, err
	}

	mult, err := parseMultiplier(cmd)
	if err != nil {
		return nil, err
	}

	return retry.NewExponentialBackoff(baseDuration, maxDuration, mult), nil
}

func parseBaseDuration(cmd *cobra.Command) (time.Duration, error) {
	baseDel := baseDelay
	if !cmd.Flags().Changed("base-delay") {
		if envBaseDelay := viper.GetString("base-delay"); envBaseDelay != "" {
			baseDel = envBaseDelay
		}
	}
	baseDuration, err := time.ParseDuration(baseDel)
	if err != nil {
		return 0, fmt.Errorf("invalid base-delay format: %w", err)
	}
	return baseDuration, nil
}

func parseMaxDuration(cmd *cobra.Command) (time.Duration, error) {
	maxDel := maxDelay
	if !cmd.Flags().Changed("max-delay") {
		if envMaxDelay := viper.GetString("max-delay"); envMaxDelay != "" {
			maxDel = envMaxDelay
		}
	}
	maxDuration, err := time.ParseDuration(maxDel)
	if err != nil {
		return 0, fmt.Errorf("invalid max-delay format: %w", err)
	}
	return maxDuration, nil
}

func parseMultiplier(cmd *cobra.Command) (float64, error) {
	mult := multiplier
	if !cmd.Flags().Changed("multiplier") {
		if envMultiplier := viper.GetFloat64("multiplier"); envMultiplier != 0 {
			mult = envMultiplier
		}
	}

	if mult <= 1.0 {
		return 0, ErrInvalidMultiplier
	}

	return mult, nil
}

func createAndRunRetry(
	commandStr string,
	finalMaxTries uint,
	backoffStrategy retry.BackoffStrategy,
	logger *slog.Logger,
) error {
	// Create retry instance
	r, err := retry.NewRetry(commandStr, retry.NewStopOnMaxTries(finalMaxTries))
	if err != nil {
		return fmt.Errorf("failed to create retry instance: %w", err)
	}

	// Set backoff strategy
	r.SetBackoffStrategy(backoffStrategy)

	// Run the retry
	err = r.Run(logger)
	if err != nil {
		return fmt.Errorf("retry failed: %w", err)
	}

	return nil
}

func parseLinearBackoff(cmd *cobra.Command) (*retry.LinearBackoff, error) {
	baseDuration, err := parseBaseDuration(cmd)
	if err != nil {
		return nil, err
	}
	
	maxDuration, err := parseMaxDuration(cmd)
	if err != nil {
		return nil, err
	}
	
	incr := increment
	if !cmd.Flags().Changed("increment") {
		if envIncrement := viper.GetString("increment"); envIncrement != "" {
			incr = envIncrement
		}
	}
	
	incrDuration, err := time.ParseDuration(incr)
	if err != nil {
		return nil, fmt.Errorf("invalid increment format: %w", err)
	}
	
	return retry.NewLinearBackoff(baseDuration, incrDuration, maxDuration), nil
}

func parseFibonacciBackoff(cmd *cobra.Command) (*retry.FibonacciBackoff, error) {
	baseDuration, err := parseBaseDuration(cmd)
	if err != nil {
		return nil, err
	}
	
	maxDuration, err := parseMaxDuration(cmd)
	if err != nil {
		return nil, err
	}
	
	return retry.NewFibonacciBackoff(baseDuration, maxDuration), nil
}

func parseCustomBackoff(cmd *cobra.Command) (*retry.CustomBackoff, error) {
	delayStr := delays
	if !cmd.Flags().Changed("delays") {
		if envDelays := viper.GetString("delays"); envDelays != "" {
			delayStr = envDelays
		}
	}
	
	if delayStr == "" {
		return nil, ErrEmptyDelays
	}
	
	// Parse comma-separated delays
	delayParts := strings.Split(delayStr, ",")
	parsedDelays := make([]time.Duration, 0, len(delayParts))
	
	for _, part := range delayParts {
		trimmed := strings.TrimSpace(part)
		duration, err := time.ParseDuration(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid delay format '%s': %w", trimmed, err)
		}
		parsedDelays = append(parsedDelays, duration)
	}
	
	return retry.NewCustomBackoff(parsedDelays), nil
}

func applyJitter(cmd *cobra.Command, strategy retry.BackoffStrategy) (*retry.JitterBackoff, error) {
	jitterValue := jitter
	if !cmd.Flags().Changed("jitter") {
		if envJitter := viper.GetFloat64("jitter"); envJitter != 0 {
			jitterValue = envJitter
		}
	}
	
	if jitterValue == 0 {
		// Return nil to indicate no jitter wrapper needed
		return nil, nil
	}
	
	if jitterValue < 0 || jitterValue > 1 {
		return nil, ErrInvalidJitter
	}
	
	return retry.NewJitterBackoff(strategy, jitterValue), nil
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