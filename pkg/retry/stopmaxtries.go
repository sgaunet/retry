package retry

import "context"

// StopOnMaxTries is a struct that implements the StopStrategy interface.
type StopOnMaxTries struct {
	maxTries uint
	tries    uint
}

// NewStopOnMaxTries returns a new StopOnMaxTries struct.
func NewStopOnMaxTries(maxTries uint) *StopOnMaxTries {
	return &StopOnMaxTries{maxTries: maxTries}
}

// GetCtx returns the background context.
func (s *StopOnMaxTries) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached returns true if the number of tries is greater than or equal to the maximum number of tries.
// If the maximum number of tries is 0, it returns false.
func (s *StopOnMaxTries) IsLimitReached() bool {
	if s.maxTries == 0 {
		return false
	}
	return s.tries >= s.maxTries
}

// StartTry increments the number of tries.
func (s *StopOnMaxTries) StartTry() {
	s.tries++
}

// EndTry does nothing.
func (s *StopOnMaxTries) EndTry() {}
