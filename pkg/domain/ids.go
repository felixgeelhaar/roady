package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// idPattern matches valid ID formats: alphanumeric with hyphens/underscores
var idPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// TaskID represents a validated task identifier.
type TaskID struct {
	value string
}

// NewTaskID creates a new TaskID from a string value.
// Returns an error if the value is invalid.
func NewTaskID(value string) (TaskID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return TaskID{}, fmt.Errorf("task ID cannot be empty")
	}
	if !idPattern.MatchString(value) {
		return TaskID{}, fmt.Errorf("invalid task ID format: %s", value)
	}
	return TaskID{value: value}, nil
}

// MustTaskID creates a TaskID or panics if invalid. Use only in tests.
func MustTaskID(value string) TaskID {
	id, err := NewTaskID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the TaskID.
func (id TaskID) String() string {
	return id.value
}

// IsZero returns true if the TaskID is empty.
func (id TaskID) IsZero() bool {
	return id.value == ""
}

// Equals checks if two TaskIDs are equal.
func (id TaskID) Equals(other TaskID) bool {
	return id.value == other.value
}

// FeatureID represents a validated feature identifier.
type FeatureID struct {
	value string
}

// NewFeatureID creates a new FeatureID from a string value.
func NewFeatureID(value string) (FeatureID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return FeatureID{}, fmt.Errorf("feature ID cannot be empty")
	}
	if !idPattern.MatchString(value) {
		return FeatureID{}, fmt.Errorf("invalid feature ID format: %s", value)
	}
	return FeatureID{value: value}, nil
}

// MustFeatureID creates a FeatureID or panics if invalid. Use only in tests.
func MustFeatureID(value string) FeatureID {
	id, err := NewFeatureID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the FeatureID.
func (id FeatureID) String() string {
	return id.value
}

// IsZero returns true if the FeatureID is empty.
func (id FeatureID) IsZero() bool {
	return id.value == ""
}

// Equals checks if two FeatureIDs are equal.
func (id FeatureID) Equals(other FeatureID) bool {
	return id.value == other.value
}

// SpecID represents a validated spec identifier.
type SpecID struct {
	value string
}

// NewSpecID creates a new SpecID from a string value.
func NewSpecID(value string) (SpecID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return SpecID{}, fmt.Errorf("spec ID cannot be empty")
	}
	if !idPattern.MatchString(value) {
		return SpecID{}, fmt.Errorf("invalid spec ID format: %s", value)
	}
	return SpecID{value: value}, nil
}

// MustSpecID creates a SpecID or panics if invalid. Use only in tests.
func MustSpecID(value string) SpecID {
	id, err := NewSpecID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the SpecID.
func (id SpecID) String() string {
	return id.value
}

// IsZero returns true if the SpecID is empty.
func (id SpecID) IsZero() bool {
	return id.value == ""
}

// Equals checks if two SpecIDs are equal.
func (id SpecID) Equals(other SpecID) bool {
	return id.value == other.value
}
