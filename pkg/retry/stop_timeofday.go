package retry

import (
	"context"
	"fmt"
	"time"
)

// StopAtTimeOfDay stops retrying at a specific time of day.
type StopAtTimeOfDay struct {
	stopTime time.Time
}

// NewStopAtTimeOfDay creates a condition that stops at a specific time.
// timeStr should be in "HH:MM" format (24-hour).
func NewStopAtTimeOfDay(timeStr string) (*StopAtTimeOfDay, error) {
	// Parse the time string (HH:MM format)
	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid time format (use HH:MM): %w", err)
	}
	
	// Get current date with specified time
	now := time.Now()
	stopTime := time.Date(now.Year(), now.Month(), now.Day(),
		parsedTime.Hour(), parsedTime.Minute(), 0, 0, now.Location())
	
	// If the time has already passed today, set it for tomorrow
	const hoursPerDay = 24
	if stopTime.Before(now) {
		stopTime = stopTime.Add(hoursPerDay * time.Hour)
	}
	
	return &StopAtTimeOfDay{
		stopTime: stopTime,
	}, nil
}

// GetCtx returns a background context as time-of-day checking doesn't need timeout.
func (s *StopAtTimeOfDay) GetCtx() context.Context {
	return context.Background()
}

// IsLimitReached checks if the current time has passed the stop time.
func (s *StopAtTimeOfDay) IsLimitReached() bool {
	return time.Now().After(s.stopTime)
}

// StartTry does nothing for time-of-day condition.
func (s *StopAtTimeOfDay) StartTry() {}

// EndTry does nothing for time-of-day condition.
func (s *StopAtTimeOfDay) EndTry() {}