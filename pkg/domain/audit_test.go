package domain

import (
	"testing"
	"time"
)

func TestEventCalculateHashDeterminism(t *testing.T) {
	event := &Event{
		ID:        "e1",
		Action:    "test.action",
		Actor:     "tester",
		Timestamp: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		PrevHash:  "prev",
	}

	first := event.CalculateHash()
	second := event.CalculateHash()
	if first != second {
		t.Fatalf("expected deterministic hash: %s vs %s", first, second)
	}

	event.ID = "e2"
	if first == event.CalculateHash() {
		t.Fatalf("hash should change when ID changes")
	}
}

func TestEventCalculateHashWithMetadata(t *testing.T) {
	base := &Event{
		ID:        "e1",
		Action:    "test.action",
		Actor:     "tester",
		Timestamp: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		PrevHash:  "prev",
	}
	baseHash := base.CalculateHash()

	// Event with metadata should have different hash
	withMeta := &Event{
		ID:        "e1",
		Action:    "test.action",
		Actor:     "tester",
		Timestamp: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		PrevHash:  "prev",
		Metadata:  map[string]interface{}{"key": "value"},
	}
	metaHash := withMeta.CalculateHash()

	if baseHash == metaHash {
		t.Fatal("hash should differ when metadata is added")
	}
}

func TestEventCalculateHashMetadataOrder(t *testing.T) {
	// Metadata with keys in different insertion order should produce same hash
	event1 := &Event{
		ID:        "e1",
		Action:    "test.action",
		Actor:     "tester",
		Timestamp: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		PrevHash:  "prev",
		Metadata: map[string]interface{}{
			"alpha": 1,
			"beta":  2,
			"gamma": 3,
		},
	}

	event2 := &Event{
		ID:        "e1",
		Action:    "test.action",
		Actor:     "tester",
		Timestamp: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		PrevHash:  "prev",
		Metadata: map[string]interface{}{
			"gamma": 3,
			"alpha": 1,
			"beta":  2,
		},
	}

	hash1 := event1.CalculateHash()
	hash2 := event2.CalculateHash()

	if hash1 != hash2 {
		t.Fatalf("hash should be same regardless of key order: %s vs %s", hash1, hash2)
	}
}

func TestCanonicalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "empty",
			input:    nil,
			expected: "",
		},
		{
			name:     "single key",
			input:    map[string]interface{}{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "multiple keys sorted",
			input:    map[string]interface{}{"z": 1, "a": 2, "m": 3},
			expected: `{"a":2,"m":3,"z":1}`,
		},
		{
			name:     "nested values",
			input:    map[string]interface{}{"outer": map[string]interface{}{"inner": "val"}},
			expected: `{"outer":{"inner":"val"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalJSON(tt.input)
			if result != tt.expected {
				t.Errorf("canonicalJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}
