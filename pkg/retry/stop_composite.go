package retry

import (
	"context"
	"reflect"
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
	// Create a context that will be cancelled when the composite is cancelled
	// OR when any timeout-based sub-condition is cancelled
	ctx, cancel := createMergedContext(conditions)

	return &CompositeCondition{
		conditions: conditions,
		logic:      logic,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// createMergedContext creates a context that gets cancelled when any sub-condition
// with a timeout context gets cancelled. This avoids goroutine leaks.
func createMergedContext(conditions []ConditionRetryer) (context.Context, context.CancelFunc) {
	// Start with a cancellable background context
	ctx, cancel := context.WithCancel(context.Background())

	// Find timeout-based conditions (those that actually use cancellable contexts)
	var timeoutCtxs []context.Context
	for _, cond := range conditions {
		condCtx := cond.GetCtx()
		// Only monitor contexts that are actually cancellable (not Background)
		if condCtx != context.Background() && condCtx != context.TODO() {
			timeoutCtxs = append(timeoutCtxs, condCtx)
		}
	}

	// If there are timeout contexts, start a single goroutine to monitor them
	if len(timeoutCtxs) > 0 {
		go func() {
			// Use a select with all timeout contexts
			cases := make([]reflect.SelectCase, len(timeoutCtxs)+1)
			for i, timeoutCtx := range timeoutCtxs {
				cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(timeoutCtx.Done())}
			}
			// Also listen for the composite context cancellation
			cases[len(timeoutCtxs)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())}

			// Wait for any context to be done
			chosen, _, _ := reflect.Select(cases)

			// Only cancel the composite context if a timeout context was triggered
			// If the composite context itself was triggered (last case), just exit
			if chosen < len(timeoutCtxs) {
				cancel()
			}
		}()
	}

	return ctx, cancel
}

// GetCtx returns the context from the composite condition.
// The composite context is automatically cancelled when any sub-condition context is cancelled.
func (c *CompositeCondition) GetCtx() context.Context {
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

// Cancel cancels the composite condition's context and recursively cancels
// all sub-conditions that support cancellation.
func (c *CompositeCondition) Cancel() {
	// Define an interface for conditions that can be cancelled
	type cancellableCondition interface {
		Cancel()
	}

	// Cancel this composite's context first
	c.cancel()

	// Recursively cancel all sub-conditions
	for _, condition := range c.conditions {
		if cancellable, ok := condition.(cancellableCondition); ok {
			cancellable.Cancel()
		}
	}
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
