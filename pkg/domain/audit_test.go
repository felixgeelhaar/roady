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
