package retry

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// Mock condition for testing
type mockCondition struct {
	ctx         context.Context
	limitReached bool
	startTryCalled int
	endTryCalled   int
}

func newMockCondition(limitReached bool) *mockCondition {
	return &mockCondition{
		ctx:         context.Background(),
		limitReached: limitReached,
	}
}

func (m *mockCondition) GetCtx() context.Context {
	return m.ctx
}

func (m *mockCondition) IsLimitReached() bool {
	return m.limitReached
}

func (m *mockCondition) StartTry() {
	m.startTryCalled++
}

func (m *mockCondition) EndTry() {
	m.endTryCalled++
}

// Mock enhanced condition for testing
type mockEnhancedCondition struct {
	*mockCondition
	lastExitCode int
	lastStdout   string
	lastStderr   string
}

func newMockEnhancedCondition(limitReached bool) *mockEnhancedCondition {
	return &mockEnhancedCondition{
		mockCondition: newMockCondition(limitReached),
		lastExitCode:  -1,
	}
}

func (m *mockEnhancedCondition) SetLastExitCode(code int) {
	m.lastExitCode = code
}

func (m *mockEnhancedCondition) SetLastOutput(stdout, stderr string) {
	m.lastStdout = stdout
	m.lastStderr = stderr
}

func TestNewCompositeCondition_AND(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(false)
	cond2 := newMockCondition(false)

	composite := NewCompositeCondition(LogicAND, cond1, cond2)
	defer composite.Cancel()

	if composite == nil {
		t.Fatal("NewCompositeCondition should return non-nil condition")
	}

	if composite.logic != LogicAND {
		t.Errorf("Expected LogicAND, got %v", composite.logic)
	}

	if len(composite.conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(composite.conditions))
	}
}

func TestNewCompositeCondition_OR(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(false)
	cond2 := newMockCondition(false)

	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	if composite.logic != LogicOR {
		t.Errorf("Expected LogicOR, got %v", composite.logic)
	}
}

func TestCompositeCondition_GetCtx_Background(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(false)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	ctx := composite.GetCtx()
	// The composite creates its own context, so it won't be exactly context.Background()
	// but it should not have an error and should not have a deadline
	if ctx.Err() != nil {
		t.Error("Context should not have an error")
	}
	if _, ok := ctx.Deadline(); ok {
		t.Error("Context should not have a deadline when no conditions have timeouts")
	}
}

func TestCompositeCondition_GetCtx_WithTimeout(t *testing.T) {
	defer goleak.VerifyNone(t)
	timeout1 := NewStopOnTimeout(1 * time.Second)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, timeout1, cond2)
	defer composite.Cancel()

	// Since the current implementation returns the first context with an error,
	// and timeout context doesn't have an error initially, it returns composite's context
	// This is actually correct behavior as the composite manages its own context
	ctx := composite.GetCtx()
	if ctx.Err() != nil {
		t.Error("Context should not have an error initially")
	}
}

func TestCompositeCondition_IsLimitReached_AND_AllFalse(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(false)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicAND, cond1, cond2)
	defer composite.Cancel()

	if composite.IsLimitReached() {
		t.Error("AND condition should return false when all conditions are false")
	}
}

func TestCompositeCondition_IsLimitReached_AND_OneFalse(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(true)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicAND, cond1, cond2)
	defer composite.Cancel()

	if composite.IsLimitReached() {
		t.Error("AND condition should return false when at least one condition is false")
	}
}

func TestCompositeCondition_IsLimitReached_AND_AllTrue(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(true)
	cond2 := newMockCondition(true)
	composite := NewCompositeCondition(LogicAND, cond1, cond2)
	defer composite.Cancel()

	if !composite.IsLimitReached() {
		t.Error("AND condition should return true when all conditions are true")
	}
}

func TestCompositeCondition_IsLimitReached_OR_AllFalse(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(false)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	if composite.IsLimitReached() {
		t.Error("OR condition should return false when all conditions are false")
	}
}

func TestCompositeCondition_IsLimitReached_OR_OneTrue(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(true)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	if !composite.IsLimitReached() {
		t.Error("OR condition should return true when at least one condition is true")
	}
}

func TestCompositeCondition_IsLimitReached_OR_AllTrue(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(true)
	cond2 := newMockCondition(true)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	if !composite.IsLimitReached() {
		t.Error("OR condition should return true when all conditions are true")
	}
}

func TestCompositeCondition_StartTry(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(false)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	composite.StartTry()

	if cond1.startTryCalled != 1 {
		t.Errorf("Expected StartTry to be called once on condition 1, got %d", cond1.startTryCalled)
	}

	if cond2.startTryCalled != 1 {
		t.Errorf("Expected StartTry to be called once on condition 2, got %d", cond2.startTryCalled)
	}
}

func TestCompositeCondition_EndTry(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(false)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	composite.EndTry()

	if cond1.endTryCalled != 1 {
		t.Errorf("Expected EndTry to be called once on condition 1, got %d", cond1.endTryCalled)
	}

	if cond2.endTryCalled != 1 {
		t.Errorf("Expected EndTry to be called once on condition 2, got %d", cond2.endTryCalled)
	}
}

func TestCompositeCondition_SetLastExitCode(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockEnhancedCondition(false)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	composite.SetLastExitCode(42)

	if cond1.lastExitCode != 42 {
		t.Errorf("Expected exit code 42 to be set on enhanced condition, got %d", cond1.lastExitCode)
	}
}

func TestCompositeCondition_SetLastOutput(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockEnhancedCondition(false)
	cond2 := newMockCondition(false)
	composite := NewCompositeCondition(LogicOR, cond1, cond2)
	defer composite.Cancel()

	composite.SetLastOutput("test stdout", "test stderr")

	if cond1.lastStdout != "test stdout" {
		t.Errorf("Expected stdout 'test stdout' to be set on enhanced condition, got %q", cond1.lastStdout)
	}

	if cond1.lastStderr != "test stderr" {
		t.Errorf("Expected stderr 'test stderr' to be set on enhanced condition, got %q", cond1.lastStderr)
	}
}

func TestCompositeCondition_EmptyConditions(t *testing.T) {
	defer goleak.VerifyNone(t)
	composite := NewCompositeCondition(LogicAND)
	defer composite.Cancel()

	// Composite creates its own context, not exactly background
	ctx := composite.GetCtx()
	if ctx.Err() != nil {
		t.Error("Empty composite should return context without error")
	}

	// AND of empty set should be true (vacuous truth)
	if !composite.IsLimitReached() {
		t.Error("AND of empty conditions should return true")
	}

	composite2 := NewCompositeCondition(LogicOR)
	defer composite2.Cancel()
	// OR of empty set should be false
	if composite2.IsLimitReached() {
		t.Error("OR of empty conditions should return false")
	}
}

func TestCompositeCondition_SingleCondition(t *testing.T) {
	defer goleak.VerifyNone(t)
	cond1 := newMockCondition(true)
	composite := NewCompositeCondition(LogicAND, cond1)
	defer composite.Cancel()

	if !composite.IsLimitReached() {
		t.Error("Single condition AND should return the condition's value")
	}

	composite2 := NewCompositeCondition(LogicOR, cond1)
	defer composite2.Cancel()
	if !composite2.IsLimitReached() {
		t.Error("Single condition OR should return the condition's value")
	}
}

func TestCompositeCondition_MixedConditionTypes(t *testing.T) {
	defer goleak.VerifyNone(t)
	// Mix regular and enhanced conditions
	timeout := NewStopOnTimeout(1 * time.Hour) // Far in future
	exitCode := NewStopOnExitCode([]int{1})
	composite := NewCompositeCondition(LogicOR, timeout, exitCode)
	defer composite.Cancel()

	// Initially both should be false
	if composite.IsLimitReached() {
		t.Error("Mixed condition OR should return false initially")
	}

	// Set exit code to trigger one condition
	composite.SetLastExitCode(1)
	if !composite.IsLimitReached() {
		t.Error("Mixed condition OR should return true when exit code matches")
	}
}