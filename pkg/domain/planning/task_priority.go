package planning

import (
	"encoding/json"
	"fmt"
)

// priorityOrder defines the ordering of priorities (higher order = higher priority)
var priorityOrder = map[TaskPriority]int{
	PriorityLow:    1,
	PriorityMedium: 2,
	PriorityHigh:   3,
}

// AllTaskPriorities returns all valid task priorities.
func AllTaskPriorities() []TaskPriority {
	return []TaskPriority{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
	}
}

// IsValid returns true if the priority is a valid task priority.
func (p TaskPriority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	default:
		return false
	}
}

// String returns the string representation of the priority.
func (p TaskPriority) String() string {
	return string(p)
}

// Order returns the numeric order of the priority (higher = more important).
func (p TaskPriority) Order() int {
	if order, ok := priorityOrder[p]; ok {
		return order
	}
	return 0
}

// Compare compares this priority to another.
// Returns -1 if p < other, 0 if p == other, 1 if p > other.
func (p TaskPriority) Compare(other TaskPriority) int {
	thisOrder := p.Order()
	otherOrder := other.Order()

	switch {
	case thisOrder < otherOrder:
		return -1
	case thisOrder > otherOrder:
		return 1
	default:
		return 0
	}
}

// IsHigherThan returns true if this priority is higher than the other.
func (p TaskPriority) IsHigherThan(other TaskPriority) bool {
	return p.Compare(other) > 0
}

// IsLowerThan returns true if this priority is lower than the other.
func (p TaskPriority) IsLowerThan(other TaskPriority) bool {
	return p.Compare(other) < 0
}

// Equals returns true if this priority equals the other.
func (p TaskPriority) Equals(other TaskPriority) bool {
	return p == other
}

// IsHigh returns true if this is high priority.
func (p TaskPriority) IsHigh() bool {
	return p == PriorityHigh
}

// IsMedium returns true if this is medium priority.
func (p TaskPriority) IsMedium() bool {
	return p == PriorityMedium
}

// IsLow returns true if this is low priority.
func (p TaskPriority) IsLow() bool {
	return p == PriorityLow
}

// DisplayName returns a human-readable display name for the priority.
func (p TaskPriority) DisplayName() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityMedium:
		return "Medium"
	case PriorityHigh:
		return "High"
	default:
		return string(p)
	}
}

// ParseTaskPriority parses a string into a TaskPriority.
func ParseTaskPriority(s string) (TaskPriority, error) {
	priority := TaskPriority(s)
	if !priority.IsValid() {
		return "", fmt.Errorf("invalid task priority: %s", s)
	}
	return priority, nil
}

// MustParseTaskPriority parses a string into a TaskPriority, panicking on error.
func MustParseTaskPriority(s string) TaskPriority {
	priority, err := ParseTaskPriority(s)
	if err != nil {
		panic(err)
	}
	return priority
}

// DefaultTaskPriority returns the default priority for new tasks.
func DefaultTaskPriority() TaskPriority {
	return PriorityMedium
}

// HighestPriority returns the highest priority from a slice of priorities.
func HighestPriority(priorities []TaskPriority) TaskPriority {
	if len(priorities) == 0 {
		return DefaultTaskPriority()
	}

	highest := priorities[0]
	for _, p := range priorities[1:] {
		if p.IsHigherThan(highest) {
			highest = p
		}
	}
	return highest
}

// LowestPriority returns the lowest priority from a slice of priorities.
func LowestPriority(priorities []TaskPriority) TaskPriority {
	if len(priorities) == 0 {
		return DefaultTaskPriority()
	}

	lowest := priorities[0]
	for _, p := range priorities[1:] {
		if p.IsLowerThan(lowest) {
			lowest = p
		}
	}
	return lowest
}

// MarshalJSON implements json.Marshaler interface.
func (p TaskPriority) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(p))
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (p *TaskPriority) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// Accept empty string as medium for backward compatibility
	if str == "" {
		*p = PriorityMedium
		return nil
	}

	priority := TaskPriority(str)
	if !priority.IsValid() {
		return fmt.Errorf("invalid task priority: %s", str)
	}

	*p = priority
	return nil
}
