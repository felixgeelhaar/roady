package domain_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain"
)

func TestTaskID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "task-1", false},
		{"valid underscore", "task_1", false},
		{"valid alphanumeric", "taskABC123", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"starts with number", "123task", true},
		{"has spaces", "task 1", true},
		{"special chars", "task@1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := domain.NewTaskID(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id.String() != tt.value {
				t.Errorf("String() = %v, want %v", id.String(), tt.value)
			}
		})
	}
}

func TestTaskID_Equals(t *testing.T) {
	id1 := domain.MustTaskID("task-1")
	id2 := domain.MustTaskID("task-1")
	id3 := domain.MustTaskID("task-2")

	if !id1.Equals(id2) {
		t.Error("expected task-1 to equal task-1")
	}
	if id1.Equals(id3) {
		t.Error("expected task-1 to not equal task-2")
	}
}

func TestTaskID_IsZero(t *testing.T) {
	var zero domain.TaskID
	if !zero.IsZero() {
		t.Error("expected zero value to be zero")
	}

	id := domain.MustTaskID("task-1")
	if id.IsZero() {
		t.Error("expected non-zero value to not be zero")
	}
}

func TestFeatureID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "feature-auth", false},
		{"empty", "", true},
		{"invalid format", "123feature", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := domain.NewFeatureID(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFeatureID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id.String() != tt.value {
				t.Errorf("String() = %v, want %v", id.String(), tt.value)
			}
		})
	}
}

func TestSpecID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "spec-v1", false},
		{"empty", "", true},
		{"invalid format", "!spec", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := domain.NewSpecID(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSpecID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id.String() != tt.value {
				t.Errorf("String() = %v, want %v", id.String(), tt.value)
			}
		})
	}
}
