package billing

import "time"

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
