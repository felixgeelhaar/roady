package events_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeEvent(eventType string, actor string, ts time.Time, meta map[string]interface{}) *events.BaseEvent {
	return &events.BaseEvent{
		ID:             "evt-" + eventType,
		Type:           eventType,
		AggregateID_:   "agg-1",
		AggregateType_: "task",
		Timestamp:      ts,
		Actor:          actor,
		Metadata:       meta,
	}
}

// ---------------------------------------------------------------------------
// TaskStateProjection
// ---------------------------------------------------------------------------

func TestTaskStateProjection_Name(t *testing.T) {
	p := events.NewTaskStateProjection()
	if got := p.Name(); got != "task_state" {
		t.Errorf("Name() = %q, want %q", got, "task_state")
	}
}

func TestTaskStateProjection_Apply(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		eventType      string
		metadata       map[string]interface{}
		wantStatus     planning.TaskStatus
		wantStartedAt  bool
		wantCompleted  bool
		wantVerified   bool
		wantBlocked    bool
		wantBlockedNil bool
		wantExtRef     bool
	}{
		{
			name:          "TaskStarted sets in_progress and StartedAt",
			eventType:     events.EventTypeTaskStarted,
			metadata:      map[string]interface{}{"task_id": "t1"},
			wantStatus:    planning.StatusInProgress,
			wantStartedAt: true,
		},
		{
			name:          "TaskCompleted sets done and CompletedAt",
			eventType:     events.EventTypeTaskCompleted,
			metadata:      map[string]interface{}{"task_id": "t1"},
			wantStatus:    planning.StatusDone,
			wantCompleted: true,
		},
		{
			name:         "TaskVerified sets verified and VerifiedAt",
			eventType:    events.EventTypeTaskVerified,
			metadata:     map[string]interface{}{"task_id": "t1"},
			wantStatus:   planning.StatusVerified,
			wantVerified: true,
		},
		{
			name:        "TaskBlocked sets blocked and BlockedAt",
			eventType:   events.EventTypeTaskBlocked,
			metadata:    map[string]interface{}{"task_id": "t1"},
			wantStatus:  planning.StatusBlocked,
			wantBlocked: true,
		},
		{
			name:           "TaskUnblocked sets pending and clears BlockedAt",
			eventType:      events.EventTypeTaskUnblocked,
			metadata:       map[string]interface{}{"task_id": "t1"},
			wantStatus:     planning.StatusPending,
			wantBlockedNil: true,
		},
		{
			name:      "TaskTransitioned uses to_status metadata",
			eventType: events.EventTypeTaskTransitioned,
			metadata: map[string]interface{}{
				"task_id":   "t1",
				"to_status": "done",
			},
			wantStatus: planning.StatusDone,
		},
		{
			name:      "ExternalRefLinked populates ExternalRefs",
			eventType: events.EventTypeExternalRefLinked,
			metadata: map[string]interface{}{
				"task_id":     "t1",
				"provider":    "github",
				"external_id": "GH-99",
				"url":         "https://github.com/org/repo/issues/99",
			},
			wantStatus: planning.StatusPending,
			wantExtRef: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := events.NewTaskStateProjection()
			evt := makeEvent(tc.eventType, "alice", now, tc.metadata)

			if err := p.Apply(evt); err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			state := p.GetState("t1")
			if state == nil {
				t.Fatal("expected state for t1, got nil")
			}

			if state.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", state.Status, tc.wantStatus)
			}

			if tc.wantStartedAt && state.StartedAt == nil {
				t.Error("expected StartedAt to be set")
			}
			if tc.wantCompleted && state.CompletedAt == nil {
				t.Error("expected CompletedAt to be set")
			}
			if tc.wantVerified && state.VerifiedAt == nil {
				t.Error("expected VerifiedAt to be set")
			}
			if tc.wantBlocked && state.BlockedAt == nil {
				t.Error("expected BlockedAt to be set")
			}
			if tc.wantBlockedNil && state.BlockedAt != nil {
				t.Error("expected BlockedAt to be nil after unblock")
			}
			if tc.wantExtRef {
				ref, ok := state.ExternalRefs["github"]
				if !ok {
					t.Fatal("expected external ref for github")
				}
				if ref.ExternalID != "GH-99" {
					t.Errorf("ExternalID = %q, want %q", ref.ExternalID, "GH-99")
				}
				if ref.URL != "https://github.com/org/repo/issues/99" {
					t.Errorf("URL = %q, want full URL", ref.URL)
				}
				if ref.Provider != "github" {
					t.Errorf("Provider = %q, want %q", ref.Provider, "github")
				}
			}
		})
	}
}

func TestTaskStateProjection_Apply_TaskTransitioned_MissingToStatus(t *testing.T) {
	p := events.NewTaskStateProjection()
	evt := makeEvent(events.EventTypeTaskTransitioned, "alice", time.Now(), map[string]interface{}{
		"task_id": "t1",
		// no "to_status" key
	})

	if err := p.Apply(evt); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	state := p.GetState("t1")
	if state == nil {
		t.Fatal("expected state for t1")
	}
	// Without to_status, status should remain the default pending.
	if state.Status != planning.StatusPending {
		t.Errorf("Status = %q, want %q (default)", state.Status, planning.StatusPending)
	}
}

func TestTaskStateProjection_Apply_UnhandledEvent(t *testing.T) {
	p := events.NewTaskStateProjection()
	evt := makeEvent(events.EventTypePlanCreated, "alice", time.Now(), nil)

	if err := p.Apply(evt); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	states := p.GetAllStates()
	if len(states) != 0 {
		t.Errorf("expected no states for unhandled event type, got %d", len(states))
	}
}

func TestTaskStateProjection_GetState(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		p := events.NewTaskStateProjection()
		evt := makeEvent(events.EventTypeTaskStarted, "bob", time.Now(), map[string]interface{}{"task_id": "t1"})
		_ = p.Apply(evt)

		state := p.GetState("t1")
		if state == nil {
			t.Fatal("expected non-nil state")
		}
		if state.TaskID != "t1" {
			t.Errorf("TaskID = %q, want %q", state.TaskID, "t1")
		}
	})

	t.Run("not found", func(t *testing.T) {
		p := events.NewTaskStateProjection()
		state := p.GetState("nonexistent")
		if state != nil {
			t.Errorf("expected nil for unknown task, got %+v", state)
		}
	})
}

func TestTaskStateProjection_GetAllStates(t *testing.T) {
	p := events.NewTaskStateProjection()
	_ = p.Apply(makeEvent(events.EventTypeTaskStarted, "a", time.Now(), map[string]interface{}{"task_id": "t1"}))
	_ = p.Apply(makeEvent(events.EventTypeTaskStarted, "b", time.Now(), map[string]interface{}{"task_id": "t2"}))
	_ = p.Apply(makeEvent(events.EventTypeTaskCompleted, "a", time.Now(), map[string]interface{}{"task_id": "t1"}))

	states := p.GetAllStates()
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states["t1"].Status != planning.StatusDone {
		t.Errorf("t1 status = %q, want %q", states["t1"].Status, planning.StatusDone)
	}
	if states["t2"].Status != planning.StatusInProgress {
		t.Errorf("t2 status = %q, want %q", states["t2"].Status, planning.StatusInProgress)
	}

	// Verify returned map is a copy (mutation does not affect projection).
	delete(states, "t1")
	if p.GetState("t1") == nil {
		t.Error("deleting from returned map should not affect projection")
	}
}

func TestTaskStateProjection_Rebuild(t *testing.T) {
	p := events.NewTaskStateProjection()
	// Apply an event directly so the projection is non-empty.
	_ = p.Apply(makeEvent(events.EventTypeTaskStarted, "a", time.Now(), map[string]interface{}{"task_id": "old"}))

	evts := []*events.BaseEvent{
		makeEvent(events.EventTypeTaskStarted, "alice", time.Now(), map[string]interface{}{"task_id": "t1"}),
		makeEvent(events.EventTypeTaskCompleted, "alice", time.Now(), map[string]interface{}{"task_id": "t1"}),
		makeEvent(events.EventTypeTaskStarted, "bob", time.Now(), map[string]interface{}{"task_id": "t2"}),
	}

	if err := p.Rebuild(evts); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	// "old" should be gone because Rebuild resets first.
	if p.GetState("old") != nil {
		t.Error("expected old state to be cleared after Rebuild")
	}

	states := p.GetAllStates()
	if len(states) != 2 {
		t.Errorf("expected 2 states after rebuild, got %d", len(states))
	}
}

func TestTaskStateProjection_Reset(t *testing.T) {
	p := events.NewTaskStateProjection()
	_ = p.Apply(makeEvent(events.EventTypeTaskStarted, "a", time.Now(), map[string]interface{}{"task_id": "t1"}))

	if err := p.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	states := p.GetAllStates()
	if len(states) != 0 {
		t.Errorf("expected 0 states after Reset, got %d", len(states))
	}
}

// ---------------------------------------------------------------------------
// VelocityProjection
// ---------------------------------------------------------------------------

func TestVelocityProjection_Name(t *testing.T) {
	p := events.NewVelocityProjection(7)
	if got := p.Name(); got != "velocity" {
		t.Errorf("Name() = %q, want %q", got, "velocity")
	}
}

func TestVelocityProjection_Apply(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		eventType     string
		wantCompCount int
		wantVerCount  int
	}{
		{
			name:          "TaskCompleted increments completions",
			eventType:     events.EventTypeTaskCompleted,
			wantCompCount: 1,
			wantVerCount:  0,
		},
		{
			name:          "TaskVerified increments verifications",
			eventType:     events.EventTypeTaskVerified,
			wantCompCount: 0,
			wantVerCount:  1,
		},
		{
			name:          "other event type is ignored",
			eventType:     events.EventTypePlanCreated,
			wantCompCount: 0,
			wantVerCount:  0,
		},
		{
			name:          "TaskStarted is ignored",
			eventType:     events.EventTypeTaskStarted,
			wantCompCount: 0,
			wantVerCount:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := events.NewVelocityProjection(30)
			evt := makeEvent(tc.eventType, "alice", now, nil)

			if err := p.Apply(evt); err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			// Velocity = count / windowDays. With 1 event in 30-day window:
			compVel := p.GetCompletionVelocity()
			verVel := p.GetVerificationVelocity()

			wantComp := float64(tc.wantCompCount) / 30.0
			wantVer := float64(tc.wantVerCount) / 30.0

			if compVel != wantComp {
				t.Errorf("GetCompletionVelocity() = %f, want %f", compVel, wantComp)
			}
			if verVel != wantVer {
				t.Errorf("GetVerificationVelocity() = %f, want %f", verVel, wantVer)
			}
		})
	}
}

func TestVelocityProjection_GetCompletionVelocity(t *testing.T) {
	t.Run("empty returns zero", func(t *testing.T) {
		p := events.NewVelocityProjection(7)
		if got := p.GetCompletionVelocity(); got != 0 {
			t.Errorf("expected 0, got %f", got)
		}
	})

	t.Run("events within window counted", func(t *testing.T) {
		p := events.NewVelocityProjection(7)
		now := time.Now()
		for i := 0; i < 7; i++ {
			evt := makeEvent(events.EventTypeTaskCompleted, "a", now.Add(-time.Duration(i)*24*time.Hour), nil)
			_ = p.Apply(evt)
		}
		got := p.GetCompletionVelocity()
		if got != 1.0 {
			t.Errorf("expected 1.0, got %f", got)
		}
	})

	t.Run("events outside window excluded", func(t *testing.T) {
		p := events.NewVelocityProjection(7)
		old := time.Now().Add(-30 * 24 * time.Hour)
		_ = p.Apply(makeEvent(events.EventTypeTaskCompleted, "a", old, nil))
		got := p.GetCompletionVelocity()
		if got != 0 {
			t.Errorf("expected 0 for old event, got %f", got)
		}
	})
}

func TestVelocityProjection_GetVerificationVelocity(t *testing.T) {
	t.Run("empty returns zero", func(t *testing.T) {
		p := events.NewVelocityProjection(7)
		if got := p.GetVerificationVelocity(); got != 0 {
			t.Errorf("expected 0, got %f", got)
		}
	})

	t.Run("tracks verification events", func(t *testing.T) {
		p := events.NewVelocityProjection(10)
		now := time.Now()
		for i := 0; i < 5; i++ {
			_ = p.Apply(makeEvent(events.EventTypeTaskVerified, "a", now.Add(-time.Duration(i)*24*time.Hour), nil))
		}
		got := p.GetVerificationVelocity()
		want := 5.0 / 10.0
		if got != want {
			t.Errorf("expected %f, got %f", want, got)
		}
	})
}

func TestVelocityProjection_DefaultWindow(t *testing.T) {
	tests := []struct {
		name       string
		windowDays int
	}{
		{"zero defaults to 7", 0},
		{"negative defaults to 7", -5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := events.NewVelocityProjection(tc.windowDays)
			// Apply 7 events within the last 7 days.
			now := time.Now()
			for i := 0; i < 7; i++ {
				_ = p.Apply(makeEvent(events.EventTypeTaskCompleted, "a", now.Add(-time.Duration(i)*24*time.Hour), nil))
			}
			// Window defaults to 7 days, so velocity should be 7/7 = 1.0.
			got := p.GetCompletionVelocity()
			if got != 1.0 {
				t.Errorf("velocity = %f, want 1.0 (default 7-day window)", got)
			}
		})
	}
}

func TestVelocityProjection_Rebuild(t *testing.T) {
	p := events.NewVelocityProjection(30)
	now := time.Now()
	_ = p.Apply(makeEvent(events.EventTypeTaskCompleted, "old", now, nil))

	evts := []*events.BaseEvent{
		makeEvent(events.EventTypeTaskCompleted, "a", now, nil),
		makeEvent(events.EventTypeTaskCompleted, "b", now, nil),
		makeEvent(events.EventTypeTaskVerified, "a", now, nil),
	}

	if err := p.Rebuild(evts); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	compVel := p.GetCompletionVelocity()
	wantComp := 2.0 / 30.0
	if compVel != wantComp {
		t.Errorf("completion velocity = %f, want %f", compVel, wantComp)
	}

	verVel := p.GetVerificationVelocity()
	wantVer := 1.0 / 30.0
	if verVel != wantVer {
		t.Errorf("verification velocity = %f, want %f", verVel, wantVer)
	}
}

func TestVelocityProjection_Reset(t *testing.T) {
	p := events.NewVelocityProjection(30)
	now := time.Now()
	_ = p.Apply(makeEvent(events.EventTypeTaskCompleted, "a", now, nil))
	_ = p.Apply(makeEvent(events.EventTypeTaskVerified, "a", now, nil))

	if err := p.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	if got := p.GetCompletionVelocity(); got != 0 {
		t.Errorf("completion velocity after reset = %f, want 0", got)
	}
	if got := p.GetVerificationVelocity(); got != 0 {
		t.Errorf("verification velocity after reset = %f, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// AuditTimelineProjection
// ---------------------------------------------------------------------------

func TestAuditTimelineProjection_Name(t *testing.T) {
	p := events.NewAuditTimelineProjection()
	if got := p.Name(); got != "audit_timeline" {
		t.Errorf("Name() = %q, want %q", got, "audit_timeline")
	}
}

func TestAuditTimelineProjection_Apply(t *testing.T) {
	now := time.Now()
	eventTypes := []string{
		events.EventTypePlanCreated,
		events.EventTypePlanApproved,
		events.EventTypeTaskStarted,
		events.EventTypeDriftDetected,
		events.EventTypeSyncCompleted,
	}

	p := events.NewAuditTimelineProjection()
	for _, et := range eventTypes {
		meta := map[string]interface{}{}
		if et == events.EventTypeTaskStarted {
			meta["task_id"] = "t1"
		}
		if et == events.EventTypeSyncCompleted {
			meta["provider"] = "github"
		}
		evt := makeEvent(et, "alice", now, meta)
		if err := p.Apply(evt); err != nil {
			t.Fatalf("Apply(%s) error = %v", et, err)
		}
	}

	timeline := p.GetTimeline()
	if len(timeline) != len(eventTypes) {
		t.Fatalf("expected %d entries, got %d", len(eventTypes), len(timeline))
	}

	for i, entry := range timeline {
		if entry.EventType != eventTypes[i] {
			t.Errorf("entry[%d].EventType = %q, want %q", i, entry.EventType, eventTypes[i])
		}
		if entry.Actor != "alice" {
			t.Errorf("entry[%d].Actor = %q, want %q", i, entry.Actor, "alice")
		}
		if entry.Description == "" {
			t.Errorf("entry[%d].Description is empty for %s", i, eventTypes[i])
		}
	}
}

func TestAuditTimelineProjection_Apply_SetsAggregateID(t *testing.T) {
	p := events.NewAuditTimelineProjection()
	evt := &events.BaseEvent{
		ID:           "e1",
		Type:         events.EventTypePlanCreated,
		AggregateID_: "plan-42",
		Timestamp:    time.Now(),
		Actor:        "bob",
	}
	_ = p.Apply(evt)

	tl := p.GetTimeline()
	if len(tl) != 1 {
		t.Fatal("expected 1 entry")
	}
	if tl[0].AggregateID != "plan-42" {
		t.Errorf("AggregateID = %q, want %q", tl[0].AggregateID, "plan-42")
	}
}

func TestAuditTimelineProjection_GetTimeline(t *testing.T) {
	p := events.NewAuditTimelineProjection()
	// Empty timeline.
	if got := p.GetTimeline(); len(got) != 0 {
		t.Errorf("expected empty timeline, got %d entries", len(got))
	}

	// After applying events.
	_ = p.Apply(makeEvent(events.EventTypePlanCreated, "a", time.Now(), nil))
	_ = p.Apply(makeEvent(events.EventTypePlanApproved, "b", time.Now(), nil))

	tl := p.GetTimeline()
	if len(tl) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tl))
	}

	// Returned slice should be a copy.
	tl[0].Actor = "MUTATED"
	original := p.GetTimeline()
	if original[0].Actor == "MUTATED" {
		t.Error("GetTimeline should return a copy")
	}
}

func TestAuditTimelineProjection_GetRecentEntries(t *testing.T) {
	p := events.NewAuditTimelineProjection()
	now := time.Now()
	for i := 0; i < 5; i++ {
		_ = p.Apply(makeEvent(events.EventTypePlanCreated, "a", now.Add(time.Duration(i)*time.Minute), nil))
	}

	t.Run("n less than total", func(t *testing.T) {
		recent := p.GetRecentEntries(3)
		if len(recent) != 3 {
			t.Fatalf("expected 3, got %d", len(recent))
		}
		// Should be the last 3 entries.
		if recent[0].Timestamp.Before(now.Add(2 * time.Minute)) {
			t.Error("expected entries from position 2 onwards")
		}
	})

	t.Run("n greater than total", func(t *testing.T) {
		recent := p.GetRecentEntries(100)
		if len(recent) != 5 {
			t.Errorf("expected 5, got %d", len(recent))
		}
	})

	t.Run("n equals zero", func(t *testing.T) {
		recent := p.GetRecentEntries(0)
		// n=0: 0 >= 5 is false, so start = 5-0 = 5, result = make([]T, 0).
		if len(recent) != 0 {
			t.Errorf("expected 0 entries for n=0, got %d", len(recent))
		}
	})

	t.Run("n equals total", func(t *testing.T) {
		recent := p.GetRecentEntries(5)
		if len(recent) != 5 {
			t.Errorf("expected 5, got %d", len(recent))
		}
	})

	t.Run("empty projection", func(t *testing.T) {
		empty := events.NewAuditTimelineProjection()
		recent := empty.GetRecentEntries(3)
		if len(recent) != 0 {
			t.Errorf("expected 0 from empty projection, got %d", len(recent))
		}
	})
}

func TestAuditTimelineProjection_Rebuild(t *testing.T) {
	p := events.NewAuditTimelineProjection()
	_ = p.Apply(makeEvent(events.EventTypePlanCreated, "old", time.Now(), nil))

	evts := []*events.BaseEvent{
		makeEvent(events.EventTypeTaskStarted, "a", time.Now(), map[string]interface{}{"task_id": "t1"}),
		makeEvent(events.EventTypeTaskCompleted, "a", time.Now(), map[string]interface{}{"task_id": "t1"}),
	}

	if err := p.Rebuild(evts); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	tl := p.GetTimeline()
	if len(tl) != 2 {
		t.Errorf("expected 2 entries after rebuild, got %d", len(tl))
	}
	// Old entry should be gone.
	for _, entry := range tl {
		if entry.Actor == "old" {
			t.Error("rebuild should clear previous entries")
		}
	}
}

func TestAuditTimelineProjection_Reset(t *testing.T) {
	p := events.NewAuditTimelineProjection()
	_ = p.Apply(makeEvent(events.EventTypePlanCreated, "a", time.Now(), nil))

	if err := p.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	if got := p.GetTimeline(); len(got) != 0 {
		t.Errorf("expected empty timeline after Reset, got %d entries", len(got))
	}
}

// ---------------------------------------------------------------------------
// describeEvent (exercised through AuditTimelineProjection.Apply)
// ---------------------------------------------------------------------------

func TestDescribeEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    *events.BaseEvent
		wantDesc string
	}{
		{
			name:     "PlanCreated",
			event:    makeEvent(events.EventTypePlanCreated, "a", time.Now(), nil),
			wantDesc: "Plan created",
		},
		{
			name:     "PlanApproved",
			event:    makeEvent(events.EventTypePlanApproved, "a", time.Now(), nil),
			wantDesc: "Plan approved",
		},
		{
			name:     "PlanRejected",
			event:    makeEvent(events.EventTypePlanRejected, "a", time.Now(), nil),
			wantDesc: "Plan rejected",
		},
		{
			name:     "TaskStarted",
			event:    makeEvent(events.EventTypeTaskStarted, "a", time.Now(), map[string]interface{}{"task_id": "t1"}),
			wantDesc: "Task started: t1",
		},
		{
			name:     "TaskCompleted",
			event:    makeEvent(events.EventTypeTaskCompleted, "a", time.Now(), map[string]interface{}{"task_id": "t2"}),
			wantDesc: "Task completed: t2",
		},
		{
			name:     "TaskVerified",
			event:    makeEvent(events.EventTypeTaskVerified, "a", time.Now(), map[string]interface{}{"task_id": "t3"}),
			wantDesc: "Task verified: t3",
		},
		{
			name:     "TaskBlocked",
			event:    makeEvent(events.EventTypeTaskBlocked, "a", time.Now(), map[string]interface{}{"task_id": "t4"}),
			wantDesc: "Task blocked: t4",
		},
		{
			name:     "TaskUnblocked",
			event:    makeEvent(events.EventTypeTaskUnblocked, "a", time.Now(), map[string]interface{}{"task_id": "t5"}),
			wantDesc: "Task unblocked: t5",
		},
		{
			name:     "SyncCompleted",
			event:    makeEvent(events.EventTypeSyncCompleted, "a", time.Now(), map[string]interface{}{"provider": "jira"}),
			wantDesc: "Sync completed with jira",
		},
		{
			name:     "DriftDetected",
			event:    makeEvent(events.EventTypeDriftDetected, "a", time.Now(), nil),
			wantDesc: "Drift detected",
		},
		{
			name:     "DriftResolved",
			event:    makeEvent(events.EventTypeDriftResolved, "a", time.Now(), nil),
			wantDesc: "Drift resolved",
		},
		{
			name:     "unknown event falls through to default",
			event:    makeEvent("custom.event", "a", time.Now(), nil),
			wantDesc: "custom.event",
		},
		{
			name:     "DriftAccepted falls through to default",
			event:    makeEvent(events.EventTypeDriftAccepted, "a", time.Now(), nil),
			wantDesc: events.EventTypeDriftAccepted,
		},
		{
			name:     "TaskTransitioned falls through to default",
			event:    makeEvent(events.EventTypeTaskTransitioned, "a", time.Now(), map[string]interface{}{"task_id": "t1", "to_status": "done"}),
			wantDesc: events.EventTypeTaskTransitioned,
		},
		{
			name:     "ExternalRefLinked falls through to default",
			event:    makeEvent(events.EventTypeExternalRefLinked, "a", time.Now(), map[string]interface{}{"task_id": "t1", "provider": "gh", "external_id": "1"}),
			wantDesc: events.EventTypeExternalRefLinked,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := events.NewAuditTimelineProjection()
			_ = p.Apply(tc.event)

			tl := p.GetTimeline()
			if len(tl) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(tl))
			}
			if tl[0].Description != tc.wantDesc {
				t.Errorf("Description = %q, want %q", tl[0].Description, tc.wantDesc)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getStringMetadata (exercised indirectly through describeEvent and Apply)
// ---------------------------------------------------------------------------

func TestGetStringMetadata(t *testing.T) {
	t.Run("key exists with string value", func(t *testing.T) {
		p := events.NewAuditTimelineProjection()
		evt := makeEvent(events.EventTypeSyncCompleted, "a", time.Now(), map[string]interface{}{
			"provider": "linear",
		})
		_ = p.Apply(evt)
		tl := p.GetTimeline()
		if tl[0].Description != "Sync completed with linear" {
			t.Errorf("expected provider to be extracted, got %q", tl[0].Description)
		}
	})

	t.Run("key missing returns empty string", func(t *testing.T) {
		p := events.NewAuditTimelineProjection()
		evt := makeEvent(events.EventTypeSyncCompleted, "a", time.Now(), map[string]interface{}{})
		_ = p.Apply(evt)
		tl := p.GetTimeline()
		// describeEvent calls getStringMetadata("provider") on empty map => ""
		if tl[0].Description != "Sync completed with " {
			t.Errorf("expected empty provider, got %q", tl[0].Description)
		}
	})

	t.Run("key exists with wrong type returns empty string", func(t *testing.T) {
		p := events.NewAuditTimelineProjection()
		evt := makeEvent(events.EventTypeSyncCompleted, "a", time.Now(), map[string]interface{}{
			"provider": 42, // int, not string
		})
		_ = p.Apply(evt)
		tl := p.GetTimeline()
		// getStringMetadata returns "" for non-string value.
		if tl[0].Description != "Sync completed with " {
			t.Errorf("expected empty provider for non-string, got %q", tl[0].Description)
		}
	})
}
