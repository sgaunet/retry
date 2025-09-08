// Package main implements a command line tool to retry a command
package main

import (
	"errors"
	"fmt"
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
	ErrInvalidConditionLogic = errors.New("must be 'and' or 'or'")
	ErrUnsupportedBackoff = errors.New(
		"unsupported backoff strategy (supported: fixed, exponential, linear, fibonacci, custom)")
	ErrInvalidJitter         = errors.New("jitter must be between 0.0 and 1.0")
	ErrEmptyDelays           = errors.New("delays cannot be empty when using custom backoff")
	ErrInvalidLogLevel       = errors.New("log level must be one of: error, warn, info, debug")
	ErrConflictingOutputModes = errors.New(
		"cannot combine --json with other output modes (--summary-only, --quiet-retries)")
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
	
	// New stop condition flags.
	timeout              string
	stopOnExit           string
	stopWhenContains     string
	stopWhenNotContains  string
	stopAt               string
	conditionLogic       string
	
	// Success/Failure condition flags (Issue #22).
	retryOnExit      string
	successOnExit    string
	retryIfContains  string
	successContains  string
	failIfContains   string
	successRegex     string
	retryRegex       string
	
	// Output control flags.
	quietRetries  bool
	noColor       bool
	summaryOnly   bool
	verboseOutput bool
	
	// New enhanced output flags.
	quiet     bool
	jsonMode  bool
	logFile   string
	logLevel  string
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
  retry --backoff exponential --jitter 0.2 "command"
  
  # Multiple stop conditions
  retry --max-tries 10 --timeout 5m "slow-command"
  
  # Stop on specific exit codes
  retry --stop-on-exit "0,1" "command"
  
  # Stop when output contains pattern
  retry --stop-when-contains "ready" --timeout 30s "service-check"
  
  # Enhanced Output Control & Logging (Issue #21)
  # JSON output for automation
  retry --json "curl -f https://api.example.com"
  
  # Quiet mode for minimal output
  retry --quiet "make test"
  
  # File logging
  retry --log-file retry.log "important-command"
  
  # Debug logging level
  retry --log-level debug "troublesome-command"
  
  # Success/Failure Conditions (Issue #22)
  # Only retry on specific exit codes
  retry --retry-on-exit "1,2,124" "flaky-service"
  
  # Consider exit code 2 as success
  retry --success-on-exit "0,2" "special-command"
  
  # Retry if output contains temporary error
  retry --retry-if-contains "temporary error" "api-call"
  
  # Success if output contains 200 OK
  retry --success-contains "200 OK" "curl https://api.example.com"
  
  # Fail immediately on fatal error
  retry --fail-if-contains "fatal error" "deployment"
  
  # Success based on regex match
  retry --success-regex "HTTP/1.1 [23][0-9][0-9]" "curl -I https://api.com"`,
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

func setupBasicFlags() {
	rootCmd.Flags().UintVarP(&maxTries, "max-tries", "t", defaultMaxTries,
		"maximum number of retry attempts (0 for infinite)")
	rootCmd.Flags().StringVarP(&delay, "delay", "d", "0s", "delay between retries (e.g., 1s, 500ms, 2m)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

func setupBackoffFlags() {
	rootCmd.Flags().StringVarP(&backoff, "backoff", "B", "fixed",
		"backoff strategy (fixed, exponential, linear, fibonacci, custom)")
	rootCmd.Flags().StringVarP(&baseDelay, "base-delay", "b", "1s", "base delay for backoff strategies")
	rootCmd.Flags().StringVarP(&maxDelay, "max-delay", "M", "5m", "maximum delay cap for backoff strategies")
	rootCmd.Flags().Float64Var(&multiplier, "multiplier", defaultMultiplier, "multiplier for exponential backoff")
	rootCmd.Flags().StringVar(&increment, "increment", "500ms", "increment for linear backoff")
	rootCmd.Flags().Float64VarP(&jitter, "jitter", "j", 0.0, "jitter percentage (0.0-1.0) to add randomness")
	rootCmd.Flags().StringVar(&delays, "delays", "", "comma-separated custom delays (e.g., 1s,2s,5s,10s)")
}

func setupStopConditionFlags() {
	rootCmd.Flags().StringVar(&timeout, "timeout", "", "stop after duration (e.g., 5m, 30s)")
	rootCmd.Flags().StringVar(&stopOnExit, "stop-on-exit", "", "stop on specific exit codes (comma-separated)")
	rootCmd.Flags().StringVar(&stopWhenContains, "stop-when-contains", "", "stop when output contains pattern")
	rootCmd.Flags().StringVar(&stopWhenNotContains, "stop-when-not-contains", "",
		"stop when output doesn't contain pattern")
	rootCmd.Flags().StringVar(&stopAt, "stop-at", "", "stop at specific time (HH:MM format)")
	rootCmd.Flags().StringVar(&conditionLogic, "condition-logic", "OR", "logic for multiple conditions (AND or OR)")
}

func setupSuccessFailureFlags() {
	// Exit code conditions
	rootCmd.Flags().StringVar(&retryOnExit, "retry-on-exit", "", 
		"only retry on specific exit codes (comma-separated)")
	rootCmd.Flags().StringVar(&successOnExit, "success-on-exit", "", 
		"consider these exit codes as success (comma-separated)")
	
	// Pattern-based conditions
	rootCmd.Flags().StringVar(&retryIfContains, "retry-if-contains", "", 
		"retry if output contains pattern")
	rootCmd.Flags().StringVar(&successContains, "success-contains", "", 
		"success if output contains pattern")
	rootCmd.Flags().StringVar(&failIfContains, "fail-if-contains", "", 
		"fail immediately if pattern found")
	
	// Regex conditions
	rootCmd.Flags().StringVar(&successRegex, "success-regex", "", 
		"success if output matches regex")
	rootCmd.Flags().StringVar(&retryRegex, "retry-regex", "", 
		"retry if output matches regex")
}

func setupOutputFlags() {
	rootCmd.Flags().BoolVar(&quietRetries, "quiet-retries", false, "only show command output on final attempt")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.Flags().BoolVar(&summaryOnly, "summary-only", false, "only show final summary")
	rootCmd.Flags().BoolVarP(&verboseOutput, "verbose-output", "V", false, "show detailed timing and condition info")
	
	// New enhanced output flags from issue #21
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "minimal output (only show final result)")
	rootCmd.Flags().BoolVar(&jsonMode, "json", false, "output results as JSON")
	rootCmd.Flags().StringVar(&logFile, "log-file", "", "write logs to file")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "set log level (error, warn, info, debug)")
}

func setupEnvironmentBindings() {
	viper.SetEnvPrefix("RETRY")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}

func bindFlagsToViper() {
	flags := []string{
		"max-tries", "delay", "verbose", "backoff", "base-delay", "max-delay",
		"multiplier", "increment", "jitter", "delays", "timeout", "stop-on-exit",
		"stop-when-contains", "stop-when-not-contains", "stop-at", "condition-logic",
		"retry-on-exit", "success-on-exit", "retry-if-contains", "success-contains",
		"fail-if-contains", "success-regex", "retry-regex",
		"quiet-retries", "no-color", "summary-only", "verbose-output",
		"quiet", "json", "log-file", "log-level",
	}
	
	for _, flag := range flags {
		_ = viper.BindPFlag(flag, rootCmd.Flags().Lookup(flag))
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	
	setupBasicFlags()
	setupBackoffFlags()
	setupStopConditionFlags()
	setupSuccessFailureFlags()
	setupOutputFlags()
	
	setupEnvironmentBindings()
	bindFlagsToViper()
}

func runRetry(cmd *cobra.Command, args []string) error {
	// Get command from positional arguments
	commandStr := strings.Join(args, " ")
	if commandStr == "" {
		return ErrCommandEmpty
	}
	
	// Validate flag combinations
	err := validateFlags(cmd)
	if err != nil {
		return err
	}

	// Parse configuration
	finalMaxTries := parseMaxTries(cmd)

	// Create enhanced logger based on flags
	enhancedLogger := createEnhancedLogger(cmd)
	defer func() {
		closeErr := enhancedLogger.Close()
		if closeErr != nil {
			// Don't override the main error, just log to stderr
			_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to close logger: %v\n", closeErr)
		}
	}()

	// Create and run retry with enhanced logging
	return createAndRunRetryWithEnhancedLogging(commandStr, finalMaxTries, cmd, enhancedLogger)
}

// validateFlags validates flag combinations and values.
func validateFlags(cmd *cobra.Command) error {
	if err := validateLogLevel(cmd); err != nil {
		return err
	}
	
	return validateOutputModesConflicts(cmd)
}

// validateLogLevel validates the log level flag value.
func validateLogLevel(cmd *cobra.Command) error {
	finalLogLevel := logLevel
	if !cmd.Flags().Changed("log-level") {
		if envLogLevel := viper.GetString("log-level"); envLogLevel != "" {
			finalLogLevel = envLogLevel
		}
	}
	
	validLevels := []string{"error", "warn", "warning", "info", "debug"}
	for _, level := range validLevels {
		if strings.ToLower(finalLogLevel) == level {
			return nil
		}
	}
	
	return ErrInvalidLogLevel
}

// validateOutputModesConflicts checks for conflicting output mode flags.
func validateOutputModesConflicts(cmd *cobra.Command) error {
	conflictCount := 0
	
	if isOutputModeEnabled(cmd, "json", jsonMode) {
		conflictCount++
	}
	if isOutputModeEnabled(cmd, "summary-only", summaryOnly) {
		conflictCount++
	}
	if isOutputModeEnabled(cmd, "quiet-retries", quietRetries) {
		conflictCount++
	}
	
	if conflictCount > 1 {
		return ErrConflictingOutputModes
	}
	
	return nil
}

// isOutputModeEnabled checks if a specific output mode flag is enabled.
func isOutputModeEnabled(cmd *cobra.Command, flagName string, flagValue bool) bool {
	return flagValue || (!cmd.Flags().Changed(flagName) && viper.GetBool(flagName))
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


// parseLogLevel converts string log level to LogLevel enum.
func parseLogLevel(levelStr string) retry.LogLevel {
	switch strings.ToLower(levelStr) {
	case "error":
		return retry.LogLevelError
	case "warn", "warning":
		return retry.LogLevelWarn
	case "info":
		return retry.LogLevelInfo
	case "debug":
		return retry.LogLevelDebug
	default:
		return retry.LogLevelInfo // Default to info
	}
}

func determineLogLevel(cmd *cobra.Command) retry.LogLevel {
	// Check for new --quiet flag (highest priority for minimal output)
	if quiet || (!cmd.Flags().Changed("quiet") && viper.GetBool("quiet")) {
		return retry.LogLevelError // Only show errors in quiet mode
	}
	
	// Check for new --log-level flag
	finalLogLevel := logLevel
	if !cmd.Flags().Changed("log-level") {
		if envLogLevel := viper.GetString("log-level"); envLogLevel != "" {
			finalLogLevel = envLogLevel
		}
	}
	
	// Parse the log level from string
	level := parseLogLevel(finalLogLevel)
	
	// Handle backward compatibility flags
	switch {
	case summaryOnly || quietRetries:
		return retry.LogLevelQuiet
	case verboseOutput:
		return retry.LogLevelVerbose
	case verbose:
		return retry.LogLevelVerbose
	}
	
	return level
}

func determineOutputMode(cmd *cobra.Command) retry.OutputMode {
	// Check for JSON mode first (highest priority)
	if jsonMode || (!cmd.Flags().Changed("json") && viper.GetBool("json")) {
		return retry.OutputModeJSON
	}
	
	// Check other output modes
	if summaryOnly || (!cmd.Flags().Changed("summary-only") && viper.GetBool("summary-only")) {
		return retry.OutputModeSummaryOnly
	} else if quietRetries || (!cmd.Flags().Changed("quiet-retries") && viper.GetBool("quiet-retries")) {
		return retry.OutputModeQuietRetries
	}
	return retry.OutputModeNormal
}

func applyEnvironmentOverrides(cmd *cobra.Command) (retry.LogLevel, retry.OutputMode, string) {
	level := determineLogLevel(cmd)
	mode := determineOutputMode(cmd)
	
	// Handle log file
	finalLogFile := logFile
	if !cmd.Flags().Changed("log-file") {
		if envLogFile := viper.GetString("log-file"); envLogFile != "" {
			finalLogFile = envLogFile
		}
	}
	
	// Apply environment overrides for other flags
	if !cmd.Flags().Changed("no-color") && viper.GetBool("no-color") {
		noColor = true
	}
	
	return level, mode, finalLogFile
}

func createEnhancedLogger(cmd *cobra.Command) *retry.Logger {
	level, mode, finalLogFile := applyEnvironmentOverrides(cmd)
	return retry.NewLoggerWithFile(level, mode, noColor, finalLogFile)
}

func createAndRunRetryWithEnhancedLogging(
	commandStr string,
	finalMaxTries uint,
	cmd *cobra.Command,
	logger *retry.Logger,
) error {
	// Build stop conditions
	condition, err := buildStopConditions(cmd, finalMaxTries)
	if err != nil {
		return fmt.Errorf("failed to build stop conditions: %w", err)
	}

	// Separate success conditions from stop conditions
	stopConditions, successConditions := separateConditions(condition)
	
	// Create retry instance
	r, err := retry.NewRetry(commandStr, stopConditions)
	if err != nil {
		return fmt.Errorf("failed to create retry instance: %w", err)
	}
	
	// Set success conditions separately
	r.SetSuccessConditions(successConditions)

	// Build strategy
	strategy, err := buildStrategy(cmd)
	if err != nil {
		return err
	}
	
	// Set backoff strategy and run with enhanced logging
	r.SetBackoffStrategy(strategy)
	err = r.RunWithEnhancedLogger(logger)
	if err != nil {
		return fmt.Errorf("retry failed: %w", err)
	}

	return nil
}

//nolint:ireturn // Strategy pattern requires interface return for polymorphism
func buildStrategy(cmd *cobra.Command) (retry.BackoffStrategy, error) {
	backoffType := getBackoffType(cmd)
	var strategy retry.BackoffStrategy
	var err error
	
	switch backoffType {
	case "fixed":
		strategy, err = parseFixedBackoff(cmd)
	case "exponential", "exp":
		strategy, err = parseExponentialBackoff(cmd)
	case "linear":
		strategy, err = parseLinearBackoff(cmd)
	case "fibonacci", "fib":
		strategy, err = parseFibonacciBackoff(cmd)
	case "custom":
		strategy, err = parseCustomBackoff(cmd)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedBackoff, backoffType)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create backoff strategy: %w", err)
	}
	
	return applyJitter(cmd, strategy)
}

//nolint:ireturn // May wrap strategy with jitter, requires interface return
func applyJitter(cmd *cobra.Command, strategy retry.BackoffStrategy) (retry.BackoffStrategy, error) {
	jitterValue := getJitterValue(cmd)
	if jitterValue == 0 {
		return strategy, nil
	}
	
	if jitterValue < 0 || jitterValue > 1 {
		return nil, ErrInvalidJitter
	}
	
	return retry.NewJitterBackoff(strategy, jitterValue), nil
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

func getJitterValue(cmd *cobra.Command) float64 {
	jitterValue := jitter
	if !cmd.Flags().Changed("jitter") {
		if envJitter := viper.GetFloat64("jitter"); envJitter != 0 {
			jitterValue = envJitter
		}
	}
	return jitterValue
}


//nolint:ireturn // Factory function needs to return interface
func buildStopConditions(cmd *cobra.Command, maxTries uint) (retry.ConditionRetryer, error) {
	conditions, err := collectConditions(cmd, maxTries)
	if err != nil {
		return nil, err
	}

	logic, err := validateAndGetConditionLogic(cmd)
	if err != nil {
		return nil, err
	}

	return createFinalCondition(conditions, logic), nil
}

func collectConditions(cmd *cobra.Command, maxTries uint) ([]retry.ConditionRetryer, error) {
	var conditions []retry.ConditionRetryer
	
	// Always add max tries condition if specified
	if maxTries > 0 {
		conditions = append(conditions, retry.NewStopOnMaxTries(maxTries))
	}
	
	// Add timeout condition
	timeoutCondition, err := addTimeoutCondition(cmd)
	if err != nil {
		return nil, err
	}
	if timeoutCondition != nil {
		conditions = append(conditions, timeoutCondition)
	}
	
	// Add exit code condition
	exitCondition, err := addExitCodeCondition(cmd)
	if err != nil {
		return nil, err
	}
	if exitCondition != nil {
		conditions = append(conditions, exitCondition)
	}
	
	// Add output conditions
	outputConditions, err := addOutputConditions(cmd)
	if err != nil {
		return nil, err
	}
	conditions = append(conditions, outputConditions...)
	
	// Add time-of-day condition
	timeCondition, err := addTimeOfDayCondition(cmd)
	if err != nil {
		return nil, err
	}
	if timeCondition != nil {
		conditions = append(conditions, timeCondition)
	}
	
	// Add success/failure conditions (Issue #22)
	successFailureConditions, err := addSuccessFailureConditions(cmd)
	if err != nil {
		return nil, err
	}
	conditions = append(conditions, successFailureConditions...)
	
	return conditions, nil
}

//nolint:ireturn // Factory function needs to return interface
func addTimeoutCondition(cmd *cobra.Command) (retry.ConditionRetryer, error) {
	timeoutValue := timeout
	if timeout != "" && !cmd.Flags().Changed("timeout") {
		timeoutValue = viper.GetString("timeout")
	}
	if timeoutValue == "" {
		return nil, nil //nolint:nilnil // Valid for optional condition creation
	}
	
	duration, err := time.ParseDuration(timeoutValue)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout format: %w", err)
	}
	return retry.NewStopOnTimeout(duration), nil
}

//nolint:ireturn // Factory function needs to return interface
func addExitCodeCondition(cmd *cobra.Command) (retry.ConditionRetryer, error) {
	exitCodes := stopOnExit
	if stopOnExit != "" && !cmd.Flags().Changed("stop-on-exit") {
		exitCodes = viper.GetString("stop-on-exit")
	}
	if exitCodes == "" {
		return nil, nil //nolint:nilnil // Valid for optional condition creation
	}
	
	codes, err := parseExitCodes(exitCodes)
	if err != nil {
		return nil, err
	}
	return retry.NewStopOnExitCode(codes), nil
}

func addOutputConditions(cmd *cobra.Command) ([]retry.ConditionRetryer, error) {
	var conditions []retry.ConditionRetryer
	
	// Add output contains condition
	containsPattern := stopWhenContains
	if stopWhenContains != "" && !cmd.Flags().Changed("stop-when-contains") {
		containsPattern = viper.GetString("stop-when-contains")
	}
	if containsPattern != "" {
		condition, err := retry.NewStopOnOutputContains(containsPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to create output contains condition: %w", err)
		}
		conditions = append(conditions, condition)
	}
	
	// Add output not contains condition
	notContainsPattern := stopWhenNotContains
	if stopWhenNotContains != "" && !cmd.Flags().Changed("stop-when-not-contains") {
		notContainsPattern = viper.GetString("stop-when-not-contains")
	}
	if notContainsPattern != "" {
		condition, err := retry.NewStopOnOutputNotContains(notContainsPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to create output not-contains condition: %w", err)
		}
		conditions = append(conditions, condition)
	}
	
	return conditions, nil
}

//nolint:ireturn // Factory function needs to return interface
func addTimeOfDayCondition(cmd *cobra.Command) (retry.ConditionRetryer, error) {
	timeOfDay := stopAt
	if stopAt != "" && !cmd.Flags().Changed("stop-at") {
		timeOfDay = viper.GetString("stop-at")
	}
	if timeOfDay == "" {
		return nil, nil //nolint:nilnil // Valid for optional condition creation
	}
	
	condition, err := retry.NewStopAtTimeOfDay(timeOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to create time-of-day condition: %w", err)
	}
	return condition, nil
}

func validateAndGetConditionLogic(cmd *cobra.Command) (retry.LogicOperator, error) {
	logic := conditionLogic
	if conditionLogic != "" && !cmd.Flags().Changed("condition-logic") {
		logic = viper.GetString("condition-logic")
	}
	
	if logic != "" {
		upperLogic := strings.ToUpper(logic)
		if upperLogic != "AND" && upperLogic != "OR" {
			return retry.LogicOR, fmt.Errorf("invalid condition logic '%s': %w", logic, ErrInvalidConditionLogic)
		}
		if upperLogic == "AND" {
			return retry.LogicAND, nil
		}
	}
	
	return retry.LogicOR, nil
}

//nolint:ireturn // Factory function needs to return interface
func createFinalCondition(conditions []retry.ConditionRetryer, logic retry.LogicOperator) retry.ConditionRetryer {
	if len(conditions) == 0 {
		// Default to max tries = 3 if no conditions specified
		const defaultMaxTries = 3
		return retry.NewStopOnMaxTries(defaultMaxTries)
	} else if len(conditions) == 1 {
		return conditions[0]
	}
	
	// Multiple conditions - use composite
	return retry.NewCompositeCondition(logic, conditions...)
}

func parseExitCodes(codesStr string) ([]int, error) {
	parts := strings.Split(codesStr, ",")
	codes := make([]int, 0, len(parts))
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var code int
		_, err := fmt.Sscanf(part, "%d", &code)
		if err != nil {
			return nil, fmt.Errorf("invalid exit code '%s': %w", part, err)
		}
		codes = append(codes, code)
	}
	
	return codes, nil
}

func addSuccessFailureConditions(cmd *cobra.Command) ([]retry.ConditionRetryer, error) {
	var conditions []retry.ConditionRetryer
	
	// Handle exit code conditions
	exitCodeConditions, err := addExitCodeConditions(cmd)
	if err != nil {
		return nil, err
	}
	conditions = append(conditions, exitCodeConditions...)
	
	// Handle pattern-based conditions
	patternConditions, err := addPatternConditions(cmd)
	if err != nil {
		return nil, err
	}
	conditions = append(conditions, patternConditions...)
	
	// Handle regex conditions
	regexConditions, err := addRegexConditions(cmd)
	if err != nil {
		return nil, err
	}
	conditions = append(conditions, regexConditions...)
	
	return conditions, nil
}

func addExitCodeConditions(cmd *cobra.Command) ([]retry.ConditionRetryer, error) {
	var conditions []retry.ConditionRetryer
	
	// Retry on exit codes
	if retryOnExitValue := getValueOrEnv(cmd, "retry-on-exit", retryOnExit); retryOnExitValue != "" {
		codes, err := parseExitCodes(retryOnExitValue)
		if err != nil {
			return nil, fmt.Errorf("invalid retry-on-exit codes: %w", err)
		}
		conditions = append(conditions, retry.NewRetryOnExitCode(codes))
	}
	
	// Success on exit codes
	if successOnExitValue := getValueOrEnv(cmd, "success-on-exit", successOnExit); successOnExitValue != "" {
		codes, err := parseExitCodes(successOnExitValue)
		if err != nil {
			return nil, fmt.Errorf("invalid success-on-exit codes: %w", err)
		}
		conditions = append(conditions, retry.NewSuccessOnExitCode(codes))
	}
	
	return conditions, nil
}

func addPatternConditions(cmd *cobra.Command) ([]retry.ConditionRetryer, error) {
	var conditions []retry.ConditionRetryer
	
	// Retry if contains
	if retryIfContainsValue := getValueOrEnv(cmd, "retry-if-contains", retryIfContains); retryIfContainsValue != "" {
		condition, err := retry.NewRetryIfContains(retryIfContainsValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry-if-contains condition: %w", err)
		}
		conditions = append(conditions, condition)
	}
	
	// Success contains
	if successContainsValue := getValueOrEnv(cmd, "success-contains", successContains); successContainsValue != "" {
		condition, err := retry.NewSuccessContains(successContainsValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create success-contains condition: %w", err)
		}
		conditions = append(conditions, condition)
	}
	
	// Fail if contains
	if failIfContainsValue := getValueOrEnv(cmd, "fail-if-contains", failIfContains); failIfContainsValue != "" {
		condition, err := retry.NewFailIfContains(failIfContainsValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create fail-if-contains condition: %w", err)
		}
		conditions = append(conditions, condition)
	}
	
	return conditions, nil
}

func addRegexConditions(cmd *cobra.Command) ([]retry.ConditionRetryer, error) {
	var conditions []retry.ConditionRetryer
	
	// Success regex
	if successRegexValue := getValueOrEnv(cmd, "success-regex", successRegex); successRegexValue != "" {
		condition, err := retry.NewSuccessRegex(successRegexValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create success-regex condition: %w", err)
		}
		conditions = append(conditions, condition)
	}
	
	// Retry regex
	if retryRegexValue := getValueOrEnv(cmd, "retry-regex", retryRegex); retryRegexValue != "" {
		condition, err := retry.NewRetryRegex(retryRegexValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry-regex condition: %w", err)
		}
		conditions = append(conditions, condition)
	}
	
	return conditions, nil
}

func getValueOrEnv(cmd *cobra.Command, flagName string, flagValue string) string {
	// If the flag was explicitly changed, use the flag value
	if cmd.Flags().Changed(flagName) {
		return flagValue
	}
	// Otherwise, check for environment variable via viper
	if envValue := viper.GetString(flagName); envValue != "" {
		return envValue
	}
	// Fall back to the flag value (which might be empty)
	return flagValue
}

func separateConditions(condition retry.ConditionRetryer) (retry.ConditionRetryer, []retry.ConditionRetryer) {
	// If it's a composite condition, separate success from stop conditions
	if comp, ok := condition.(*retry.CompositeCondition); ok {
		return separateCompositeConditions(comp)
	}
	
	// If it's a single success condition, return it as a success condition
	if isSuccessCondition(condition) {
		// Return a default stop condition (max tries = 1) and the success condition
		return retry.NewStopOnMaxTries(1), []retry.ConditionRetryer{condition}
	}
	
	// Otherwise, it's a stop condition
	return condition, nil
}

func separateCompositeConditions(comp *retry.CompositeCondition) (retry.ConditionRetryer, []retry.ConditionRetryer) {
	var stopConditions []retry.ConditionRetryer
	var successConditions []retry.ConditionRetryer
	
	for _, cond := range comp.GetConditions() {
		if isSuccessCondition(cond) {
			successConditions = append(successConditions, cond)
		} else {
			stopConditions = append(stopConditions, cond)
		}
	}
	
	// Build final stop condition
	var finalStopCondition retry.ConditionRetryer
	if len(stopConditions) == 0 {
		// No stop conditions, use default
		finalStopCondition = retry.NewStopOnMaxTries(1)
	} else if len(stopConditions) == 1 {
		finalStopCondition = stopConditions[0]
	} else {
		// Multiple stop conditions, recreate composite
		finalStopCondition = retry.NewCompositeCondition(retry.LogicOR, stopConditions...)
	}
	
	return finalStopCondition, successConditions
}

func isSuccessCondition(condition retry.ConditionRetryer) bool {
	switch condition.(type) {
	case *retry.SuccessOnExitCode, *retry.SuccessContains, *retry.SuccessRegex:
		return true
	default:
		return false
	}
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