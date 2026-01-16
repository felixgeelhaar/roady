package planning

import (
	"encoding/json"
	"fmt"
)

// AllApprovalStatuses returns all valid approval statuses.
func AllApprovalStatuses() []ApprovalStatus {
	return []ApprovalStatus{
		ApprovalPending,
		ApprovalApproved,
		ApprovalRejected,
	}
}

// IsValid returns true if the status is a valid approval status.
func (s ApprovalStatus) IsValid() bool {
	switch s {
	case ApprovalPending, ApprovalApproved, ApprovalRejected:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
func (s ApprovalStatus) String() string {
	return string(s)
}

// IsPending returns true if the status is pending.
func (s ApprovalStatus) IsPending() bool {
	return s == ApprovalPending
}

// IsApproved returns true if the status is approved.
func (s ApprovalStatus) IsApproved() bool {
	return s == ApprovalApproved
}

// IsRejected returns true if the status is rejected.
func (s ApprovalStatus) IsRejected() bool {
	return s == ApprovalRejected
}

// IsFinal returns true if the status is in a final state (approved or rejected).
func (s ApprovalStatus) IsFinal() bool {
	return s == ApprovalApproved || s == ApprovalRejected
}

// CanTransitionTo returns true if a transition to the target status is allowed.
func (s ApprovalStatus) CanTransitionTo(target ApprovalStatus) bool {
	switch s {
	case ApprovalPending:
		// From pending, can go to approved or rejected
		return target == ApprovalApproved || target == ApprovalRejected
	case ApprovalApproved:
		// From approved, can go back to pending (re-review)
		return target == ApprovalPending
	case ApprovalRejected:
		// From rejected, can go back to pending (re-submit)
		return target == ApprovalPending
	default:
		return false
	}
}

// ValidTransitions returns all valid target statuses from this status.
func (s ApprovalStatus) ValidTransitions() []ApprovalStatus {
	switch s {
	case ApprovalPending:
		return []ApprovalStatus{ApprovalApproved, ApprovalRejected}
	case ApprovalApproved:
		return []ApprovalStatus{ApprovalPending}
	case ApprovalRejected:
		return []ApprovalStatus{ApprovalPending}
	default:
		return nil
	}
}

// DisplayName returns a human-readable display name for the status.
func (s ApprovalStatus) DisplayName() string {
	switch s {
	case ApprovalPending:
		return "Pending"
	case ApprovalApproved:
		return "Approved"
	case ApprovalRejected:
		return "Rejected"
	default:
		return string(s)
	}
}

// ParseApprovalStatus parses a string into an ApprovalStatus.
func ParseApprovalStatus(str string) (ApprovalStatus, error) {
	status := ApprovalStatus(str)
	if !status.IsValid() {
		return "", fmt.Errorf("invalid approval status: %s", str)
	}
	return status, nil
}

// MustParseApprovalStatus parses a string into an ApprovalStatus, panicking on error.
func MustParseApprovalStatus(str string) ApprovalStatus {
	status, err := ParseApprovalStatus(str)
	if err != nil {
		panic(err)
	}
	return status
}

// DefaultApprovalStatus returns the default approval status for new plans.
func DefaultApprovalStatus() ApprovalStatus {
	return ApprovalPending
}

// MarshalJSON implements json.Marshaler interface.
func (s ApprovalStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (s *ApprovalStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// Accept empty string as pending for backward compatibility
	if str == "" {
		*s = ApprovalPending
		return nil
	}

	status := ApprovalStatus(str)
	if !status.IsValid() {
		return fmt.Errorf("invalid approval status: %s", str)
	}

	*s = status
	return nil
}
