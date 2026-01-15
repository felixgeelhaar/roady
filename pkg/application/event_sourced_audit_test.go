package application

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestEventSourcedAuditService_Log(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}
	publisher := storage.NewInMemoryEventPublisher()

	svc, err := NewEventSourcedAuditService(store, publisher)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Log an event
	err = svc.Log("task.started", "alice", map[string]interface{}{
		"task_id": "task-1",
	})
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify event was stored
	evts, err := svc.LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents failed: %v", err)
	}
	if len(evts) != 1 {
		t.Errorf("Expected 1 event, got %d", len(evts))
	}
	if evts[0].Type != "task.started" {
		t.Errorf("Expected task.started, got %s", evts[0].Type)
	}
	if evts[0].Actor != "alice" {
		t.Errorf("Expected alice, got %s", evts[0].Actor)
	}
}

func TestEventSourcedAuditService_Projections(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}
	publisher := storage.NewInMemoryEventPublisher()

	svc, err := NewEventSourcedAuditService(store, publisher)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Log task events
	svc.Log(events.EventTypeTaskStarted, "alice", map[string]interface{}{
		"task_id": "task-1",
	})
	svc.Log(events.EventTypeTaskCompleted, "alice", map[string]interface{}{
		"task_id": "task-1",
	})

	// Check timeline projection
	timeline := svc.GetTimeline()
	if len(timeline) != 2 {
		t.Errorf("Expected 2 timeline entries, got %d", len(timeline))
	}

	// Check recent timeline
	recent := svc.GetRecentTimeline(1)
	if len(recent) != 1 {
		t.Errorf("Expected 1 recent entry, got %d", len(recent))
	}
}

func TestEventSourcedAuditService_TaskState(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}
	publisher := storage.NewInMemoryEventPublisher()

	svc, err := NewEventSourcedAuditService(store, publisher)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Log task started
	svc.Log(events.EventTypeTaskStarted, "alice", map[string]interface{}{
		"task_id": "task-1",
	})

	// Check task state
	state := svc.GetTaskState("task-1")
	if state == nil {
		t.Fatal("Expected task state")
	}
	if state.Owner != "alice" {
		t.Errorf("Expected owner alice, got %s", state.Owner)
	}

	// Get all states
	states := svc.GetAllTaskStates()
	if len(states) != 1 {
		t.Errorf("Expected 1 state, got %d", len(states))
	}
}

func TestEventSourcedAuditService_Velocity(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}
	publisher := storage.NewInMemoryEventPublisher()

	svc, err := NewEventSourcedAuditService(store, publisher)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Verify starts at zero
	velocity := svc.GetCompletionVelocity()
	if velocity != 0 {
		t.Errorf("Expected 0 velocity, got %f", velocity)
	}

	// Log completions
	for i := 0; i < 7; i++ {
		svc.Log(events.EventTypeTaskCompleted, "alice", map[string]interface{}{
			"task_id": "task-" + string(rune('0'+i)),
		})
	}

	// Velocity should now be ~1 per day
	velocity = svc.GetCompletionVelocity()
	if velocity < 0.9 || velocity > 1.1 {
		t.Errorf("Expected velocity around 1.0, got %f", velocity)
	}
}

func TestEventSourcedAuditService_VerifyIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}

	svc, err := NewEventSourcedAuditService(store, nil)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Log events
	svc.Log("task.started", "alice", nil)
	svc.Log("task.completed", "alice", nil)

	// Verify integrity
	violations, err := svc.VerifyIntegrity()
	if err != nil {
		t.Fatalf("VerifyIntegrity failed: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("Expected no violations, got: %v", violations)
	}
}

func TestEventSourcedAuditService_LoadEventsSince(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}

	svc, err := NewEventSourcedAuditService(store, nil)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Record time before logging
	before := time.Now().Add(-time.Second)

	svc.Log("task.started", "alice", nil)
	svc.Log("task.completed", "alice", nil)

	// Load since before
	evts, err := svc.LoadEventsSince(before)
	if err != nil {
		t.Fatalf("LoadEventsSince failed: %v", err)
	}
	if len(evts) != 2 {
		t.Errorf("Expected 2 events, got %d", len(evts))
	}

	// Load since future
	future := time.Now().Add(time.Hour)
	evts, err = svc.LoadEventsSince(future)
	if err != nil {
		t.Fatalf("LoadEventsSince failed: %v", err)
	}
	if len(evts) != 0 {
		t.Errorf("Expected 0 events, got %d", len(evts))
	}
}

func TestEventSourcedAuditService_RebuildFromExisting(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}

	// Pre-populate events directly in store
	store.Append(&events.BaseEvent{
		Type:  events.EventTypeTaskStarted,
		Actor: "bob",
		Metadata: map[string]interface{}{
			"task_id": "task-1",
		},
	})

	// Create service - should rebuild projections
	svc, err := NewEventSourcedAuditService(store, nil)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Check projection was rebuilt
	state := svc.GetTaskState("task-1")
	if state == nil {
		t.Fatal("Expected task state from rebuilt projection")
	}
	if state.Owner != "bob" {
		t.Errorf("Expected owner bob, got %s", state.Owner)
	}
}

func TestEventSourcedAuditService_NilPublisher(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileEventStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileEventStore failed: %v", err)
	}

	// Create with nil publisher - should not panic
	svc, err := NewEventSourcedAuditService(store, nil)
	if err != nil {
		t.Fatalf("NewEventSourcedAuditService failed: %v", err)
	}

	// Log should still work (just no publish)
	err = svc.Log("test.action", "alice", nil)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	evts, _ := svc.LoadEvents()
	if len(evts) != 1 {
		t.Errorf("Expected 1 event, got %d", len(evts))
	}
}
