package planning

import (
	"encoding/json"
	"testing"
)

func TestApprovalStatus_IsValid(t *testing.T) {
	tests := []struct {
		status ApprovalStatus
		valid  bool
	}{
		{ApprovalPending, true},
		{ApprovalApproved, true},
		{ApprovalRejected, true},
		{ApprovalStatus("invalid"), false},
		{ApprovalStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestApprovalStatus_StatusHelpers(t *testing.T) {
	// Test IsPending
	if !ApprovalPending.IsPending() {
		t.Error("ApprovalPending.IsPending() should be true")
	}
	if ApprovalApproved.IsPending() {
		t.Error("ApprovalApproved.IsPending() should be false")
	}

	// Test IsApproved
	if !ApprovalApproved.IsApproved() {
		t.Error("ApprovalApproved.IsApproved() should be true")
	}
	if ApprovalPending.IsApproved() {
		t.Error("ApprovalPending.IsApproved() should be false")
	}

	// Test IsRejected
	if !ApprovalRejected.IsRejected() {
		t.Error("ApprovalRejected.IsRejected() should be true")
	}
	if ApprovalApproved.IsRejected() {
		t.Error("ApprovalApproved.IsRejected() should be false")
	}
}

func TestApprovalStatus_IsFinal(t *testing.T) {
	tests := []struct {
		status  ApprovalStatus
		isFinal bool
	}{
		{ApprovalPending, false},
		{ApprovalApproved, true},
		{ApprovalRejected, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsFinal(); got != tt.isFinal {
				t.Errorf("IsFinal() = %v, want %v", got, tt.isFinal)
			}
		})
	}
}

func TestApprovalStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from  ApprovalStatus
		to    ApprovalStatus
		canDo bool
	}{
		// From Pending
		{ApprovalPending, ApprovalApproved, true},
		{ApprovalPending, ApprovalRejected, true},
		{ApprovalPending, ApprovalPending, false},

		// From Approved
		{ApprovalApproved, ApprovalPending, true},
		{ApprovalApproved, ApprovalRejected, false},
		{ApprovalApproved, ApprovalApproved, false},

		// From Rejected
		{ApprovalRejected, ApprovalPending, true},
		{ApprovalRejected, ApprovalApproved, false},
		{ApprovalRejected, ApprovalRejected, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.canDo {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.canDo)
			}
		})
	}
}

func TestApprovalStatus_ValidTransitions(t *testing.T) {
	tests := []struct {
		status   ApprovalStatus
		expected int
	}{
		{ApprovalPending, 2},  // approved, rejected
		{ApprovalApproved, 1}, // pending
		{ApprovalRejected, 1}, // pending
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.ValidTransitions()
			if len(got) != tt.expected {
				t.Errorf("len(ValidTransitions()) = %d, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestApprovalStatus_DisplayName(t *testing.T) {
	tests := []struct {
		status  ApprovalStatus
		display string
	}{
		{ApprovalPending, "Pending"},
		{ApprovalApproved, "Approved"},
		{ApprovalRejected, "Rejected"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.DisplayName(); got != tt.display {
				t.Errorf("DisplayName() = %v, want %v", got, tt.display)
			}
		})
	}
}

func TestParseApprovalStatus(t *testing.T) {
	tests := []struct {
		input     string
		expected  ApprovalStatus
		shouldErr bool
	}{
		{"pending", ApprovalPending, false},
		{"approved", ApprovalApproved, false},
		{"rejected", ApprovalRejected, false},
		{"invalid", ApprovalStatus(""), true},
		{"", ApprovalStatus(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseApprovalStatus(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("ParseApprovalStatus() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestDefaultApprovalStatus(t *testing.T) {
	def := DefaultApprovalStatus()
	if def != ApprovalPending {
		t.Errorf("DefaultApprovalStatus() = %v, want %v", def, ApprovalPending)
	}
}

func TestApprovalStatus_JSONMarshal(t *testing.T) {
	status := ApprovalApproved

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `"approved"`
	if string(data) != expected {
		t.Errorf("Marshal = %s, want %s", string(data), expected)
	}
}

func TestApprovalStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		input    string
		expected ApprovalStatus
	}{
		{`"pending"`, ApprovalPending},
		{`"approved"`, ApprovalApproved},
		{`"rejected"`, ApprovalRejected},
		{`""`, ApprovalPending}, // backward compatibility
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var status ApprovalStatus
			if err := json.Unmarshal([]byte(tt.input), &status); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if status != tt.expected {
				t.Errorf("Unmarshal = %v, want %v", status, tt.expected)
			}
		})
	}
}

func TestApprovalStatus_JSONUnmarshal_Invalid(t *testing.T) {
	var status ApprovalStatus
	err := json.Unmarshal([]byte(`"invalid_status"`), &status)
	if err == nil {
		t.Error("Expected error for invalid status")
	}
}

func TestAllApprovalStatuses(t *testing.T) {
	statuses := AllApprovalStatuses()
	if len(statuses) != 3 {
		t.Errorf("len(AllApprovalStatuses()) = %d, want 3", len(statuses))
	}

	expected := map[ApprovalStatus]bool{
		ApprovalPending:  false,
		ApprovalApproved: false,
		ApprovalRejected: false,
	}

	for _, s := range statuses {
		expected[s] = true
	}

	for s, found := range expected {
		if !found {
			t.Errorf("Missing status in AllApprovalStatuses: %s", s)
		}
	}
}

func TestApprovalStatus_String(t *testing.T) {
	if ApprovalApproved.String() != "approved" {
		t.Errorf("String() = %v, want approved", ApprovalApproved.String())
	}
}

func TestMustParseApprovalStatus_Valid(t *testing.T) {
	status := MustParseApprovalStatus("approved")
	if status != ApprovalApproved {
		t.Errorf("MustParseApprovalStatus() = %v, want %v", status, ApprovalApproved)
	}
}

func TestMustParseApprovalStatus_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid status")
		}
	}()
	MustParseApprovalStatus("invalid")
}

func TestApprovalStatus_CanTransitionTo_InvalidStatus(t *testing.T) {
	invalid := ApprovalStatus("bogus")
	if invalid.CanTransitionTo(ApprovalPending) {
		t.Error("expected false for invalid source status")
	}
}

func TestApprovalStatus_ValidTransitions_InvalidStatus(t *testing.T) {
	invalid := ApprovalStatus("bogus")
	transitions := invalid.ValidTransitions()
	if transitions != nil {
		t.Errorf("expected nil transitions for invalid status, got %v", transitions)
	}
}

func TestApprovalStatus_DisplayName_InvalidStatus(t *testing.T) {
	invalid := ApprovalStatus("bogus")
	display := invalid.DisplayName()
	if display != "bogus" {
		t.Errorf("expected raw string for invalid status, got %s", display)
	}
}

func TestApprovalStatus_JSONUnmarshal_BadJSON(t *testing.T) {
	var status ApprovalStatus
	err := json.Unmarshal([]byte(`not-valid-json`), &status)
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}
