package main

// Names of the stop, success and failure condition flags. They are referenced
// when declaring flags, binding them to viper, wiring shell completion and
// reporting evaluated conditions, so they are kept in a single place.
const (
	flagTimeout             = "timeout"
	flagStopOnExit          = "stop-on-exit"
	flagStopWhenContains    = "stop-when-contains"
	flagStopWhenNotContains = "stop-when-not-contains"
	flagStopAt              = "stop-at"
	flagRetryOnExit         = "retry-on-exit"
	flagSuccessOnExit       = "success-on-exit"
	flagRetryIfContains     = "retry-if-contains"
	flagSuccessContains     = "success-contains"
	flagFailIfContains      = "fail-if-contains"
	flagSuccessRegex        = "success-regex"
	flagRetryRegex          = "retry-regex"
)

// jsonTimeout identifies the timeout condition in JSON output: it is the
// condition key, the stop condition name and the status reported on timeout.
const jsonTimeout = "timeout"
