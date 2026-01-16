package planning

import (
	"encoding/json"
	"fmt"
)

// validTransitions defines the allowed state transitions and their events.
// Map: currentStatus -> event -> targetStatus
var validTransitions = map[TaskStatus]map[string]TaskStatus{
	StatusPending: {
		"start": StatusInProgress,
		"block": StatusBlocked,
	},
	StatusInProgress: {
		"complete": StatusDone,
		"block":    StatusBlocked,
		"stop":     StatusPending,
	},
	StatusBlocked: {
		"unblock": StatusPending,
	},
	StatusDone: {
		"reopen": StatusPending,
		"verify": StatusVerified,
	},
	StatusVerified: {
		"reopen": StatusPending,
	},
}

// AllTaskStatuses returns all valid task statuses.
func AllTaskStatuses() []TaskStatus {
	return []TaskStatus{
		StatusPending,
		StatusInProgress,
		StatusBlocked,
		StatusDone,
		StatusVerified,
	}
}

// IsValid returns true if the status is a valid task status.
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusBlocked, StatusDone, StatusVerified:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
func (s TaskStatus) String() string {
	return string(s)
}

// CanTransitionTo returns true if a transition from the current status to the target is allowed.
func (s TaskStatus) CanTransitionTo(target TaskStatus) bool {
	transitions, ok := validTransitions[s]
	if !ok {
		return false
	}

	for _, t := range transitions {
		if t == target {
			return true
		}
	}
	return false
}

// CanTransitionWith returns true if the given event can trigger a transition from this status.
func (s TaskStatus) CanTransitionWith(event string) bool {
	transitions, ok := validTransitions[s]
	if !ok {
		return false
	}

	_, ok = transitions[event]
	return ok
}

// TransitionWith returns the target status for a given event, or an error if not allowed.
func (s TaskStatus) TransitionWith(event string) (TaskStatus, error) {
	transitions, ok := validTransitions[s]
	if !ok {
		return s, fmt.Errorf("no transitions defined for status: %s", s)
	}

	target, ok := transitions[event]
	if !ok {
		return s, fmt.Errorf("event '%s' not allowed from status '%s'", event, s)
	}

	return target, nil
}

// ValidTransitions returns all valid target statuses that can be reached from this status.
func (s TaskStatus) ValidTransitions() []TaskStatus {
	transitions, ok := validTransitions[s]
	if !ok {
		return nil
	}

	var targets []TaskStatus
	for _, t := range transitions {
		targets = append(targets, t)
	}
	return targets
}

// ValidEvents returns all valid events that can be triggered from this status.
func (s TaskStatus) ValidEvents() []string {
	transitions, ok := validTransitions[s]
	if !ok {
		return nil
	}

	var events []string
	for event := range transitions {
		events = append(events, event)
	}
	return events
}

// IsFinal returns true if this is a terminal status (no further work expected).
func (s TaskStatus) IsFinal() bool {
	return s == StatusVerified
}

// IsComplete returns true if the task is done or verified.
func (s TaskStatus) IsComplete() bool {
	return s == StatusDone || s == StatusVerified
}

// IsInProgress returns true if the task is currently being worked on.
func (s TaskStatus) IsInProgress() bool {
	return s == StatusInProgress
}

// IsBlocked returns true if the task is blocked.
func (s TaskStatus) IsBlocked() bool {
	return s == StatusBlocked
}

// IsPending returns true if the task hasn't started yet.
func (s TaskStatus) IsPending() bool {
	return s == StatusPending
}

// RequiresOwner returns true if this status typically requires an assigned owner.
func (s TaskStatus) RequiresOwner() bool {
	return s == StatusInProgress
}

// RequiresEvidence returns true if transitions from this status require evidence.
func (s TaskStatus) RequiresEvidence() bool {
	return s == StatusInProgress // completion evidence
}

// DisplayName returns a human-readable display name for the status.
func (s TaskStatus) DisplayName() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusInProgress:
		return "In Progress"
	case StatusBlocked:
		return "Blocked"
	case StatusDone:
		return "Done"
	case StatusVerified:
		return "Verified"
	default:
		return string(s)
	}
}

// ParseTaskStatus parses a string into a TaskStatus.
func ParseTaskStatus(s string) (TaskStatus, error) {
	status := TaskStatus(s)
	if !status.IsValid() {
		return "", fmt.Errorf("invalid task status: %s", s)
	}
	return status, nil
}

// MustParseTaskStatus parses a string into a TaskStatus, panicking on error.
func MustParseTaskStatus(s string) TaskStatus {
	status, err := ParseTaskStatus(s)
	if err != nil {
		panic(err)
	}
	return status
}

// MarshalJSON implements json.Marshaler interface.
func (s TaskStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (s *TaskStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// Accept empty string as pending for backward compatibility
	if str == "" {
		*s = StatusPending
		return nil
	}

	status := TaskStatus(str)
	if !status.IsValid() {
		return fmt.Errorf("invalid task status: %s", str)
	}

	*s = status
	return nil
}
