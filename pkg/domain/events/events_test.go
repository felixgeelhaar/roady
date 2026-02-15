package events

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestBaseEvent_CalculateHash(t *testing.T) {
	event := &BaseEvent{
		ID:             "evt-123",
		Type:           EventTypeTaskCompleted,
		AggregateID_:   "task-1",
		AggregateType_: AggregateTypeTask,
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Actor:          "alice",
		Metadata: map[string]interface{}{
			"task_id": "task-1",
		},
		PrevHash: "abc123",
	}

	hash := event.CalculateHash()
	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Hash should be deterministic
	hash2 := event.CalculateHash()
	if hash != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Changing data should change hash
	event.Actor = "bob"
	hash3 := event.CalculateHash()
	if hash == hash3 {
		t.Error("Changing data should change hash")
	}
}

func TestBaseEvent_CalculateHash_EmptyMetadata(t *testing.T) {
	event := &BaseEvent{
		ID:        "evt-123",
		Type:      EventTypeTaskStarted,
		Timestamp: time.Now(),
		Actor:     "alice",
	}

	hash := event.CalculateHash()
	if hash == "" {
		t.Error("Expected non-empty hash even with empty metadata")
	}
}

func TestCanonicalJSON(t *testing.T) {
	// Keys should be sorted
	m := map[string]interface{}{
		"zebra": 1,
		"alpha": 2,
		"beta":  3,
	}

	result := canonicalJSON(m)
	expected := `{"alpha":2,"beta":3,"zebra":1}`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCanonicalJSON_Empty(t *testing.T) {
	result := canonicalJSON(nil)
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}

	result = canonicalJSON(map[string]interface{}{})
	if result != "" {
		t.Errorf("Expected empty string for empty map, got %s", result)
	}
}

func TestDomainEventInterface(t *testing.T) {
	now := time.Now()
	event := BaseEvent{
		ID:             "evt-123",
		Type:           EventTypeTaskCompleted,
		AggregateID_:   "task-1",
		AggregateType_: AggregateTypeTask,
		Timestamp:      now,
		Version_:       1,
	}

	// Verify interface implementation
	var de DomainEvent = &event
	if de.EventType() != EventTypeTaskCompleted {
		t.Errorf("Expected %s, got %s", EventTypeTaskCompleted, de.EventType())
	}
	if de.AggregateID() != "task-1" {
		t.Errorf("Expected task-1, got %s", de.AggregateID())
	}
	if de.AggregateType() != AggregateTypeTask {
		t.Errorf("Expected %s, got %s", AggregateTypeTask, de.AggregateType())
	}
	if de.Version() != 1 {
		t.Errorf("Expected version 1, got %d", de.Version())
	}
}

func TestTaskStateProjection(t *testing.T) {
	proj := NewTaskStateProjection()

	// Apply task started event
	err := proj.Apply(&BaseEvent{
		Type:      EventTypeTaskStarted,
		Timestamp: time.Now(),
		Actor:     "alice",
		Metadata: map[string]interface{}{
			"task_id": "task-1",
		},
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	state := proj.GetState("task-1")
	if state == nil {
		t.Fatal("Expected state for task-1")
	}
	if state.Status != planning.StatusInProgress {
		t.Errorf("Expected status in_progress, got %s", state.Status)
	}
	if state.Owner != "alice" {
		t.Errorf("Expected owner alice, got %s", state.Owner)
	}

	// Apply task completed event
	err = proj.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: time.Now(),
		Actor:     "alice",
		Metadata: map[string]interface{}{
			"task_id": "task-1",
		},
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	state = proj.GetState("task-1")
	if state.Status != planning.StatusDone {
		t.Errorf("Expected status done, got %s", state.Status)
	}
	if state.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestTaskStateProjection_ExternalRef(t *testing.T) {
	proj := NewTaskStateProjection()

	err := proj.Apply(&BaseEvent{
		Type:      EventTypeExternalRefLinked,
		Timestamp: time.Now(),
		Actor:     "system",
		Metadata: map[string]interface{}{
			"task_id":     "task-1",
			"provider":    "github",
			"external_id": "123",
			"url":         "https://github.com/org/repo/issues/123",
		},
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	state := proj.GetState("task-1")
	if state == nil {
		t.Fatal("Expected state for task-1")
	}
	if len(state.ExternalRefs) != 1 {
		t.Errorf("Expected 1 external ref, got %d", len(state.ExternalRefs))
	}
	if ref, ok := state.ExternalRefs["github"]; !ok {
		t.Error("Expected github ref")
	} else if ref.ExternalID != "123" {
		t.Errorf("Expected external ID 123, got %s", ref.ExternalID)
	}
}

func TestTaskStateProjection_Rebuild(t *testing.T) {
	proj := NewTaskStateProjection()

	events := []*BaseEvent{
		{Type: EventTypeTaskStarted, Timestamp: time.Now(), Actor: "alice", Metadata: map[string]interface{}{"task_id": "task-1"}},
		{Type: EventTypeTaskCompleted, Timestamp: time.Now(), Actor: "alice", Metadata: map[string]interface{}{"task_id": "task-1"}},
		{Type: EventTypeTaskStarted, Timestamp: time.Now(), Actor: "bob", Metadata: map[string]interface{}{"task_id": "task-2"}},
	}

	err := proj.Rebuild(events)
	if err != nil {
		t.Fatalf("Rebuild failed: %v", err)
	}

	states := proj.GetAllStates()
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}
	if states["task-1"].Status != planning.StatusDone {
		t.Errorf("Expected task-1 to be done")
	}
	if states["task-2"].Status != planning.StatusInProgress {
		t.Errorf("Expected task-2 to be in_progress")
	}
}

func TestVelocityProjection(t *testing.T) {
	proj := NewVelocityProjection(7)

	// Add completions within window
	for i := 0; i < 7; i++ {
		err := proj.Apply(&BaseEvent{
			Type:      EventTypeTaskCompleted,
			Timestamp: time.Now().Add(-time.Duration(i) * 24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("Apply failed: %v", err)
		}
	}

	velocity := proj.GetCompletionVelocity()
	if velocity != 1.0 {
		t.Errorf("Expected velocity 1.0, got %f", velocity)
	}
}

func TestVelocityProjection_Empty(t *testing.T) {
	proj := NewVelocityProjection(7)

	velocity := proj.GetCompletionVelocity()
	if velocity != 0 {
		t.Errorf("Expected velocity 0 for empty projection, got %f", velocity)
	}
}

func TestAuditTimelineProjection(t *testing.T) {
	proj := NewAuditTimelineProjection()

	events := []*BaseEvent{
		{Type: EventTypePlanCreated, Timestamp: time.Now().Add(-2 * time.Hour), Actor: "ai"},
		{Type: EventTypePlanApproved, Timestamp: time.Now().Add(-1 * time.Hour), Actor: "alice"},
		{Type: EventTypeTaskStarted, Timestamp: time.Now(), Actor: "alice", Metadata: map[string]interface{}{"task_id": "task-1"}},
	}

	for _, e := range events {
		if err := proj.Apply(e); err != nil {
			t.Fatalf("Apply failed: %v", err)
		}
	}

	timeline := proj.GetTimeline()
	if len(timeline) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(timeline))
	}

	recent := proj.GetRecentEntries(2)
	if len(recent) != 2 {
		t.Errorf("Expected 2 recent entries, got %d", len(recent))
	}
}

func TestAuditTimelineProjection_Descriptions(t *testing.T) {
	proj := NewAuditTimelineProjection()

	testCases := []struct {
		event    *BaseEvent
		contains string
	}{
		{&BaseEvent{Type: EventTypePlanCreated}, "Plan created"},
		{&BaseEvent{Type: EventTypePlanApproved}, "Plan approved"},
		{&BaseEvent{Type: EventTypeTaskStarted, Metadata: map[string]interface{}{"task_id": "t1"}}, "Task started"},
		{&BaseEvent{Type: EventTypeTaskCompleted, Metadata: map[string]interface{}{"task_id": "t1"}}, "Task completed"},
		{&BaseEvent{Type: EventTypeSyncCompleted, Metadata: map[string]interface{}{"provider": "github"}}, "Sync completed"},
	}

	for _, tc := range testCases {
		proj.Reset()
		proj.Apply(tc.event)
		timeline := proj.GetTimeline()
		if len(timeline) != 1 {
			t.Fatalf("Expected 1 entry for %s", tc.event.Type)
		}
		if timeline[0].Description == "" {
			t.Errorf("Expected description for %s", tc.event.Type)
		}
	}
}
