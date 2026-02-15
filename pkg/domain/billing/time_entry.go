package billing

import (
	"fmt"
	"time"
)

// NewTimeEntry creates a validated TimeEntry.
func NewTimeEntry(id, taskID, rateID string, minutes int, description string, createdAt time.Time) (TimeEntry, error) {
	if id == "" {
		return TimeEntry{}, fmt.Errorf("time entry ID must not be empty")
	}
	if taskID == "" {
		return TimeEntry{}, fmt.Errorf("task ID must not be empty")
	}
	if rateID == "" {
		return TimeEntry{}, fmt.Errorf("rate ID must not be empty")
	}
	if minutes <= 0 {
		return TimeEntry{}, fmt.Errorf("minutes must be positive")
	}
	return TimeEntry{
		ID:          id,
		TaskID:      taskID,
		RateID:      rateID,
		Minutes:     minutes,
		Description: description,
		CreatedAt:   createdAt,
	}, nil
}

type TimeEntry struct {
	ID          string    `yaml:"id" json:"id"`
	TaskID      string    `yaml:"task_id" json:"task_id"`
	RateID      string    `yaml:"rate_id" json:"rate_id"`
	Minutes     int       `yaml:"minutes" json:"minutes"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time `yaml:"created_at" json:"created_at"`
}

func (te *TimeEntry) Hours() float64 {
	return float64(te.Minutes) / 60.0
}
