package billing

import (
	"testing"
	"time"
)

func TestNewTimeEntry(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		id          string
		taskID      string
		rateID      string
		minutes     int
		description string
		wantErr     bool
	}{
		{name: "valid entry", id: "te-1", taskID: "task-1", rateID: "rate-1", minutes: 60, description: "work", wantErr: false},
		{name: "no description", id: "te-2", taskID: "task-1", rateID: "rate-1", minutes: 30, description: "", wantErr: false},
		{name: "empty ID", id: "", taskID: "task-1", rateID: "rate-1", minutes: 60, wantErr: true},
		{name: "empty taskID", id: "te-1", taskID: "", rateID: "rate-1", minutes: 60, wantErr: true},
		{name: "empty rateID", id: "te-1", taskID: "task-1", rateID: "", minutes: 60, wantErr: true},
		{name: "zero minutes", id: "te-1", taskID: "task-1", rateID: "rate-1", minutes: 0, wantErr: true},
		{name: "negative minutes", id: "te-1", taskID: "task-1", rateID: "rate-1", minutes: -5, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := NewTimeEntry(tt.id, tt.taskID, tt.rateID, tt.minutes, tt.description, now)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if entry.ID != tt.id {
				t.Errorf("expected ID %s, got %s", tt.id, entry.ID)
			}
			if entry.TaskID != tt.taskID {
				t.Errorf("expected TaskID %s, got %s", tt.taskID, entry.TaskID)
			}
			if entry.Minutes != tt.minutes {
				t.Errorf("expected Minutes %d, got %d", tt.minutes, entry.Minutes)
			}
		})
	}
}

func TestTimeEntry_Hours(t *testing.T) {
	tests := []struct {
		name     string
		minutes  int
		expected float64
	}{
		{name: "zero minutes", minutes: 0, expected: 0},
		{name: "60 minutes", minutes: 60, expected: 1.0},
		{name: "30 minutes", minutes: 30, expected: 0.5},
		{name: "90 minutes", minutes: 90, expected: 1.5},
		{name: "45 minutes", minutes: 45, expected: 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := TimeEntry{Minutes: tt.minutes}
			result := entry.Hours()
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestTimeEntry_Structure(t *testing.T) {
	now := time.Now()
	entry := TimeEntry{
		ID:          "te-123",
		TaskID:      "task-1",
		RateID:      "senior",
		Minutes:     120,
		Description: "Working on feature X",
		CreatedAt:   now,
	}

	if entry.ID != "te-123" {
		t.Errorf("expected ID te-123 but got %s", entry.ID)
	}
	if entry.TaskID != "task-1" {
		t.Errorf("expected TaskID task-1 but got %s", entry.TaskID)
	}
	if entry.RateID != "senior" {
		t.Errorf("expected RateID senior but got %s", entry.RateID)
	}
	if entry.Minutes != 120 {
		t.Errorf("expected Minutes 120 but got %d", entry.Minutes)
	}
	if entry.Description != "Working on feature X" {
		t.Errorf("expected Description but got %s", entry.Description)
	}
	if entry.Hours() != 2.0 {
		t.Errorf("expected 2.0 hours but got %v", entry.Hours())
	}
}
