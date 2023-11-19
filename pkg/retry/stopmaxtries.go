package retry

import "context"

type StopOnMaxTries struct {
	maxTries uint
	tries    uint
}

func NewStopOnMaxTries(maxTries uint) *StopOnMaxTries {
	return &StopOnMaxTries{maxTries: maxTries}
}

func (s *StopOnMaxTries) GetCtx() context.Context {
	return context.Background()
}

func (s *StopOnMaxTries) IsLimitReached() bool {
	return s.tries >= s.maxTries
}

func (s *StopOnMaxTries) StartTry() {
	s.tries++
}

func (s *StopOnMaxTries) EndTry() {}
