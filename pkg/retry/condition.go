package retry

// EnhancedConditionRetryer extends the ConditionRetryer interface with additional methods
// for handling exit codes and output. This interface is optional - conditions can implement
// it to receive additional information about command execution.
type EnhancedConditionRetryer interface {
	ConditionRetryer
	SetLastExitCode(code int)
	SetLastOutput(stdout, stderr string)
}

// LogicOperator defines how multiple conditions are combined.
type LogicOperator string

const (
	// LogicAND requires all conditions to be met to stop.
	LogicAND LogicOperator = "AND"
	// LogicOR stops when any condition is met (default).
	LogicOR LogicOperator = "OR"
)