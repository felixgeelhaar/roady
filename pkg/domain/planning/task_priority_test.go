package planning

import (
	"encoding/json"
	"testing"
)

func TestTaskPriority_IsValid(t *testing.T) {
	tests := []struct {
		priority TaskPriority
		valid    bool
	}{
		{PriorityLow, true},
		{PriorityMedium, true},
		{PriorityHigh, true},
		{TaskPriority("invalid"), false},
		{TaskPriority(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestTaskPriority_Order(t *testing.T) {
	tests := []struct {
		priority TaskPriority
		order    int
	}{
		{PriorityLow, 1},
		{PriorityMedium, 2},
		{PriorityHigh, 3},
		{TaskPriority("invalid"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.Order(); got != tt.order {
				t.Errorf("Order() = %v, want %v", got, tt.order)
			}
		})
	}
}

func TestTaskPriority_Compare(t *testing.T) {
	tests := []struct {
		p1       TaskPriority
		p2       TaskPriority
		expected int
	}{
		{PriorityLow, PriorityLow, 0},
		{PriorityLow, PriorityMedium, -1},
		{PriorityLow, PriorityHigh, -1},
		{PriorityMedium, PriorityLow, 1},
		{PriorityMedium, PriorityMedium, 0},
		{PriorityMedium, PriorityHigh, -1},
		{PriorityHigh, PriorityLow, 1},
		{PriorityHigh, PriorityMedium, 1},
		{PriorityHigh, PriorityHigh, 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.p1)+"_vs_"+string(tt.p2), func(t *testing.T) {
			if got := tt.p1.Compare(tt.p2); got != tt.expected {
				t.Errorf("Compare() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskPriority_IsHigherThan(t *testing.T) {
	tests := []struct {
		p1       TaskPriority
		p2       TaskPriority
		expected bool
	}{
		{PriorityHigh, PriorityMedium, true},
		{PriorityHigh, PriorityLow, true},
		{PriorityMedium, PriorityLow, true},
		{PriorityMedium, PriorityHigh, false},
		{PriorityLow, PriorityMedium, false},
		{PriorityHigh, PriorityHigh, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.p1)+"_higher_than_"+string(tt.p2), func(t *testing.T) {
			if got := tt.p1.IsHigherThan(tt.p2); got != tt.expected {
				t.Errorf("IsHigherThan() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskPriority_IsLowerThan(t *testing.T) {
	tests := []struct {
		p1       TaskPriority
		p2       TaskPriority
		expected bool
	}{
		{PriorityLow, PriorityMedium, true},
		{PriorityLow, PriorityHigh, true},
		{PriorityMedium, PriorityHigh, true},
		{PriorityHigh, PriorityMedium, false},
		{PriorityMedium, PriorityLow, false},
		{PriorityLow, PriorityLow, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.p1)+"_lower_than_"+string(tt.p2), func(t *testing.T) {
			if got := tt.p1.IsLowerThan(tt.p2); got != tt.expected {
				t.Errorf("IsLowerThan() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskPriority_Equals(t *testing.T) {
	if !PriorityHigh.Equals(PriorityHigh) {
		t.Error("PriorityHigh should equal itself")
	}
	if PriorityHigh.Equals(PriorityMedium) {
		t.Error("PriorityHigh should not equal PriorityMedium")
	}
}

func TestTaskPriority_PriorityChecks(t *testing.T) {
	// Test IsHigh
	if !PriorityHigh.IsHigh() {
		t.Error("PriorityHigh.IsHigh() should be true")
	}
	if PriorityMedium.IsHigh() {
		t.Error("PriorityMedium.IsHigh() should be false")
	}

	// Test IsMedium
	if !PriorityMedium.IsMedium() {
		t.Error("PriorityMedium.IsMedium() should be true")
	}
	if PriorityHigh.IsMedium() {
		t.Error("PriorityHigh.IsMedium() should be false")
	}

	// Test IsLow
	if !PriorityLow.IsLow() {
		t.Error("PriorityLow.IsLow() should be true")
	}
	if PriorityMedium.IsLow() {
		t.Error("PriorityMedium.IsLow() should be false")
	}
}

func TestTaskPriority_DisplayName(t *testing.T) {
	tests := []struct {
		priority TaskPriority
		display  string
	}{
		{PriorityLow, "Low"},
		{PriorityMedium, "Medium"},
		{PriorityHigh, "High"},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.DisplayName(); got != tt.display {
				t.Errorf("DisplayName() = %v, want %v", got, tt.display)
			}
		})
	}
}

func TestParseTaskPriority(t *testing.T) {
	tests := []struct {
		input     string
		expected  TaskPriority
		shouldErr bool
	}{
		{"low", PriorityLow, false},
		{"medium", PriorityMedium, false},
		{"high", PriorityHigh, false},
		{"invalid", TaskPriority(""), true},
		{"", TaskPriority(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseTaskPriority(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("ParseTaskPriority() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestDefaultTaskPriority(t *testing.T) {
	def := DefaultTaskPriority()
	if def != PriorityMedium {
		t.Errorf("DefaultTaskPriority() = %v, want %v", def, PriorityMedium)
	}
}

func TestHighestPriority(t *testing.T) {
	tests := []struct {
		name       string
		priorities []TaskPriority
		expected   TaskPriority
	}{
		{"empty", []TaskPriority{}, PriorityMedium},
		{"single low", []TaskPriority{PriorityLow}, PriorityLow},
		{"single high", []TaskPriority{PriorityHigh}, PriorityHigh},
		{"mixed", []TaskPriority{PriorityLow, PriorityHigh, PriorityMedium}, PriorityHigh},
		{"all low", []TaskPriority{PriorityLow, PriorityLow}, PriorityLow},
		{"high first", []TaskPriority{PriorityHigh, PriorityLow}, PriorityHigh},
		{"high last", []TaskPriority{PriorityLow, PriorityHigh}, PriorityHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HighestPriority(tt.priorities); got != tt.expected {
				t.Errorf("HighestPriority() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLowestPriority(t *testing.T) {
	tests := []struct {
		name       string
		priorities []TaskPriority
		expected   TaskPriority
	}{
		{"empty", []TaskPriority{}, PriorityMedium},
		{"single high", []TaskPriority{PriorityHigh}, PriorityHigh},
		{"single low", []TaskPriority{PriorityLow}, PriorityLow},
		{"mixed", []TaskPriority{PriorityLow, PriorityHigh, PriorityMedium}, PriorityLow},
		{"all high", []TaskPriority{PriorityHigh, PriorityHigh}, PriorityHigh},
		{"low first", []TaskPriority{PriorityLow, PriorityHigh}, PriorityLow},
		{"low last", []TaskPriority{PriorityHigh, PriorityLow}, PriorityLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LowestPriority(tt.priorities); got != tt.expected {
				t.Errorf("LowestPriority() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskPriority_JSONMarshal(t *testing.T) {
	priority := PriorityHigh

	data, err := json.Marshal(priority)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `"high"`
	if string(data) != expected {
		t.Errorf("Marshal = %s, want %s", string(data), expected)
	}
}

func TestTaskPriority_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		input    string
		expected TaskPriority
	}{
		{`"low"`, PriorityLow},
		{`"medium"`, PriorityMedium},
		{`"high"`, PriorityHigh},
		{`""`, PriorityMedium}, // backward compatibility
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var priority TaskPriority
			if err := json.Unmarshal([]byte(tt.input), &priority); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if priority != tt.expected {
				t.Errorf("Unmarshal = %v, want %v", priority, tt.expected)
			}
		})
	}
}

func TestTaskPriority_JSONUnmarshal_Invalid(t *testing.T) {
	var priority TaskPriority
	err := json.Unmarshal([]byte(`"invalid_priority"`), &priority)
	if err == nil {
		t.Error("Expected error for invalid priority")
	}
}

func TestAllTaskPriorities(t *testing.T) {
	priorities := AllTaskPriorities()
	if len(priorities) != 3 {
		t.Errorf("len(AllTaskPriorities()) = %d, want 3", len(priorities))
	}

	expected := map[TaskPriority]bool{
		PriorityLow:    false,
		PriorityMedium: false,
		PriorityHigh:   false,
	}

	for _, p := range priorities {
		expected[p] = true
	}

	for p, found := range expected {
		if !found {
			t.Errorf("Missing priority in AllTaskPriorities: %s", p)
		}
	}
}

func TestTaskPriority_String(t *testing.T) {
	if PriorityHigh.String() != "high" {
		t.Errorf("String() = %v, want high", PriorityHigh.String())
	}
}
