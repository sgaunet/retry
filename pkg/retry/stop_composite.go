package retry

import (
	"context"
)

// CompositeCondition combines multiple stop conditions with AND/OR logic.
type CompositeCondition struct {
	conditions []ConditionRetryer
	logic      LogicOperator
	ctx        context.Context //nolint:containedctx // Required for composite condition management
	cancel     context.CancelFunc
}

// NewCompositeCondition creates a new composite condition with the specified logic.
func NewCompositeCondition(logic LogicOperator, conditions ...ConditionRetryer) *CompositeCondition {
	ctx, cancel := context.WithCancel(context.Background())
	return &CompositeCondition{
		conditions: conditions,
		logic:      logic,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// GetCtx returns the context from the composite condition.
// It returns the first context that has an error, or the composite's own context.
func (c *CompositeCondition) GetCtx() context.Context {
	// Check if any sub-condition has a context error
	for _, condition := range c.conditions {
		if ctx := condition.GetCtx(); ctx.Err() != nil {
			return ctx
		}
	}
	return c.ctx
}

// IsLimitReached checks if the composite condition has been met based on the logic operator.
func (c *CompositeCondition) IsLimitReached() bool {
	if c.logic == LogicAND {
		// For AND logic, all conditions must be met
		for _, condition := range c.conditions {
			// Skip success conditions when checking stop limits
			if c.isSuccessCondition(condition) {
				continue
			}
			if !condition.IsLimitReached() {
				return false
			}
		}
		return true
	}
	
	// For OR logic (default), any condition being met stops retry
	for _, condition := range c.conditions {
		// Skip success conditions when checking stop limits
		if c.isSuccessCondition(condition) {
			continue
		}
		if condition.IsLimitReached() {
			return true
		}
	}
	return false
}


// StartTry calls StartTry on all sub-conditions.
func (c *CompositeCondition) StartTry() {
	for _, condition := range c.conditions {
		condition.StartTry()
	}
}

// EndTry calls EndTry on all sub-conditions.
func (c *CompositeCondition) EndTry() {
	for _, condition := range c.conditions {
		condition.EndTry()
	}
}

// SetLastExitCode passes the exit code to all enhanced conditions.
func (c *CompositeCondition) SetLastExitCode(code int) {
	for _, condition := range c.conditions {
		if enhanced, ok := condition.(EnhancedConditionRetryer); ok {
			enhanced.SetLastExitCode(code)
		}
	}
}

// SetLastOutput passes the output to all enhanced conditions.
func (c *CompositeCondition) SetLastOutput(stdout, stderr string) {
	for _, condition := range c.conditions {
		if enhanced, ok := condition.(EnhancedConditionRetryer); ok {
			enhanced.SetLastOutput(stdout, stderr)
		}
	}
}

// Cancel cancels the composite condition's context.
func (c *CompositeCondition) Cancel() {
	c.cancel()
}

// GetConditions returns the list of conditions for checking success conditions.
func (c *CompositeCondition) GetConditions() []ConditionRetryer {
	return c.conditions
}

// isSuccessCondition checks if a condition is a success-type condition.
func (c *CompositeCondition) isSuccessCondition(condition ConditionRetryer) bool {
	switch condition.(type) {
	case *SuccessOnExitCode, *SuccessContains, *SuccessRegex:
		return true
	default:
		return false
	}
}