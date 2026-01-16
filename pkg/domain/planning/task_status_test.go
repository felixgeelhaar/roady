package planning

import (
	"encoding/json"
	"testing"
)

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		status TaskStatus
		valid  bool
	}{
		{StatusPending, true},
		{StatusInProgress, true},
		{StatusBlocked, true},
		{StatusDone, true},
		{StatusVerified, true},
		{TaskStatus("invalid"), false},
		{TaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestTaskStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from   TaskStatus
		to     TaskStatus
		canDo  bool
	}{
		// From Pending
		{StatusPending, StatusInProgress, true},
		{StatusPending, StatusBlocked, true},
		{StatusPending, StatusDone, false},
		{StatusPending, StatusVerified, false},

		// From InProgress
		{StatusInProgress, StatusDone, true},
		{StatusInProgress, StatusBlocked, true},
		{StatusInProgress, StatusPending, true},
		{StatusInProgress, StatusVerified, false},

		// From Blocked
		{StatusBlocked, StatusPending, true},
		{StatusBlocked, StatusInProgress, false},
		{StatusBlocked, StatusDone, false},

		// From Done
		{StatusDone, StatusPending, true},
		{StatusDone, StatusVerified, true},
		{StatusDone, StatusInProgress, false},

		// From Verified
		{StatusVerified, StatusPending, true},
		{StatusVerified, StatusDone, false},
		{StatusVerified, StatusInProgress, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.canDo {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.canDo)
			}
		})
	}
}

func TestTaskStatus_CanTransitionWith(t *testing.T) {
	tests := []struct {
		status TaskStatus
		event  string
		canDo  bool
	}{
		{StatusPending, "start", true},
		{StatusPending, "block", true},
		{StatusPending, "complete", false},
		{StatusInProgress, "complete", true},
		{StatusInProgress, "stop", true},
		{StatusBlocked, "unblock", true},
		{StatusDone, "verify", true},
		{StatusDone, "reopen", true},
		{StatusVerified, "reopen", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status)+"_"+tt.event, func(t *testing.T) {
			if got := tt.status.CanTransitionWith(tt.event); got != tt.canDo {
				t.Errorf("CanTransitionWith(%s) = %v, want %v", tt.event, got, tt.canDo)
			}
		})
	}
}

func TestTaskStatus_TransitionWith(t *testing.T) {
	tests := []struct {
		status    TaskStatus
		event     string
		expected  TaskStatus
		shouldErr bool
	}{
		{StatusPending, "start", StatusInProgress, false},
		{StatusPending, "complete", StatusPending, true},
		{StatusInProgress, "complete", StatusDone, false},
		{StatusBlocked, "unblock", StatusPending, false},
		{StatusDone, "verify", StatusVerified, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status)+"_"+tt.event, func(t *testing.T) {
			got, err := tt.status.TransitionWith(tt.event)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("TransitionWith() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestTaskStatus_ValidTransitions(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected int
	}{
		{StatusPending, 2},     // start, block
		{StatusInProgress, 3},  // complete, block, stop
		{StatusBlocked, 1},     // unblock
		{StatusDone, 2},        // reopen, verify
		{StatusVerified, 1},    // reopen
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

func TestTaskStatus_ValidEvents(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected []string
	}{
		{StatusPending, []string{"start", "block"}},
		{StatusInProgress, []string{"complete", "block", "stop"}},
		{StatusBlocked, []string{"unblock"}},
		{StatusDone, []string{"reopen", "verify"}},
		{StatusVerified, []string{"reopen"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.ValidEvents()
			if len(got) != len(tt.expected) {
				t.Errorf("len(ValidEvents()) = %d, want %d", len(got), len(tt.expected))
			}
		})
	}
}

func TestTaskStatus_IsFinal(t *testing.T) {
	tests := []struct {
		status TaskStatus
		isFinal bool
	}{
		{StatusPending, false},
		{StatusInProgress, false},
		{StatusBlocked, false},
		{StatusDone, false},
		{StatusVerified, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsFinal(); got != tt.isFinal {
				t.Errorf("IsFinal() = %v, want %v", got, tt.isFinal)
			}
		})
	}
}

func TestTaskStatus_IsComplete(t *testing.T) {
	tests := []struct {
		status     TaskStatus
		isComplete bool
	}{
		{StatusPending, false},
		{StatusInProgress, false},
		{StatusBlocked, false},
		{StatusDone, true},
		{StatusVerified, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsComplete(); got != tt.isComplete {
				t.Errorf("IsComplete() = %v, want %v", got, tt.isComplete)
			}
		})
	}
}

func TestTaskStatus_RequiresOwner(t *testing.T) {
	tests := []struct {
		status        TaskStatus
		requiresOwner bool
	}{
		{StatusPending, false},
		{StatusInProgress, true},
		{StatusBlocked, false},
		{StatusDone, false},
		{StatusVerified, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.RequiresOwner(); got != tt.requiresOwner {
				t.Errorf("RequiresOwner() = %v, want %v", got, tt.requiresOwner)
			}
		})
	}
}

func TestTaskStatus_DisplayName(t *testing.T) {
	tests := []struct {
		status  TaskStatus
		display string
	}{
		{StatusPending, "Pending"},
		{StatusInProgress, "In Progress"},
		{StatusBlocked, "Blocked"},
		{StatusDone, "Done"},
		{StatusVerified, "Verified"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.DisplayName(); got != tt.display {
				t.Errorf("DisplayName() = %v, want %v", got, tt.display)
			}
		})
	}
}

func TestParseTaskStatus(t *testing.T) {
	tests := []struct {
		input     string
		expected  TaskStatus
		shouldErr bool
	}{
		{"pending", StatusPending, false},
		{"in_progress", StatusInProgress, false},
		{"blocked", StatusBlocked, false},
		{"done", StatusDone, false},
		{"verified", StatusVerified, false},
		{"invalid", TaskStatus(""), true},
		{"", TaskStatus(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseTaskStatus(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("ParseTaskStatus() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestTaskStatus_JSONMarshal(t *testing.T) {
	status := StatusInProgress

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `"in_progress"`
	if string(data) != expected {
		t.Errorf("Marshal = %s, want %s", string(data), expected)
	}
}

func TestTaskStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		input    string
		expected TaskStatus
	}{
		{`"pending"`, StatusPending},
		{`"in_progress"`, StatusInProgress},
		{`"blocked"`, StatusBlocked},
		{`"done"`, StatusDone},
		{`"verified"`, StatusVerified},
		{`""`, StatusPending}, // backward compatibility
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var status TaskStatus
			if err := json.Unmarshal([]byte(tt.input), &status); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if status != tt.expected {
				t.Errorf("Unmarshal = %v, want %v", status, tt.expected)
			}
		})
	}
}

func TestTaskStatus_JSONUnmarshal_Invalid(t *testing.T) {
	var status TaskStatus
	err := json.Unmarshal([]byte(`"invalid_status"`), &status)
	if err == nil {
		t.Error("Expected error for invalid status")
	}
}

func TestAllTaskStatuses(t *testing.T) {
	statuses := AllTaskStatuses()
	if len(statuses) != 5 {
		t.Errorf("len(AllTaskStatuses()) = %d, want 5", len(statuses))
	}

	expected := map[TaskStatus]bool{
		StatusPending:    false,
		StatusInProgress: false,
		StatusBlocked:    false,
		StatusDone:       false,
		StatusVerified:   false,
	}

	for _, s := range statuses {
		expected[s] = true
	}

	for s, found := range expected {
		if !found {
			t.Errorf("Missing status in AllTaskStatuses: %s", s)
		}
	}
}

func TestTaskStatus_StatusHelpers(t *testing.T) {
	// Test IsInProgress
	if !StatusInProgress.IsInProgress() {
		t.Error("StatusInProgress.IsInProgress() should be true")
	}
	if StatusPending.IsInProgress() {
		t.Error("StatusPending.IsInProgress() should be false")
	}

	// Test IsBlocked
	if !StatusBlocked.IsBlocked() {
		t.Error("StatusBlocked.IsBlocked() should be true")
	}
	if StatusDone.IsBlocked() {
		t.Error("StatusDone.IsBlocked() should be false")
	}

	// Test IsPending
	if !StatusPending.IsPending() {
		t.Error("StatusPending.IsPending() should be true")
	}
	if StatusDone.IsPending() {
		t.Error("StatusDone.IsPending() should be false")
	}
}
