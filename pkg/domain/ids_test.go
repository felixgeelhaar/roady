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

func TestMustTaskID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid task ID")
		}
	}()
	domain.MustTaskID("")
}

func TestMustFeatureID_Valid(t *testing.T) {
	id := domain.MustFeatureID("feature-auth")
	if id.String() != "feature-auth" {
		t.Errorf("expected feature-auth, got %s", id.String())
	}
}

func TestMustFeatureID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid feature ID")
		}
	}()
	domain.MustFeatureID("")
}

func TestFeatureID_IsZero(t *testing.T) {
	var zero domain.FeatureID
	if !zero.IsZero() {
		t.Error("expected zero value to be zero")
	}

	id := domain.MustFeatureID("feature-auth")
	if id.IsZero() {
		t.Error("expected non-zero value to not be zero")
	}
}

func TestFeatureID_Equals(t *testing.T) {
	id1 := domain.MustFeatureID("feature-auth")
	id2 := domain.MustFeatureID("feature-auth")
	id3 := domain.MustFeatureID("feature-billing")

	if !id1.Equals(id2) {
		t.Error("expected equal feature IDs to be equal")
	}
	if id1.Equals(id3) {
		t.Error("expected different feature IDs to not be equal")
	}
}

func TestMustSpecID_Valid(t *testing.T) {
	id := domain.MustSpecID("spec-v1")
	if id.String() != "spec-v1" {
		t.Errorf("expected spec-v1, got %s", id.String())
	}
}

func TestMustSpecID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid spec ID")
		}
	}()
	domain.MustSpecID("")
}

func TestSpecID_IsZero(t *testing.T) {
	var zero domain.SpecID
	if !zero.IsZero() {
		t.Error("expected zero value to be zero")
	}

	id := domain.MustSpecID("spec-v1")
	if id.IsZero() {
		t.Error("expected non-zero value to not be zero")
	}
}

func TestSpecID_Equals(t *testing.T) {
	id1 := domain.MustSpecID("spec-v1")
	id2 := domain.MustSpecID("spec-v1")
	id3 := domain.MustSpecID("spec-v2")

	if !id1.Equals(id2) {
		t.Error("expected equal spec IDs to be equal")
	}
	if id1.Equals(id3) {
		t.Error("expected different spec IDs to not be equal")
	}
}
