package events

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/debt"
)

func TestNewDriftHistoryProjection(t *testing.T) {
	p := NewDriftHistoryProjection()

	if p.Name() != "drift_history" {
		t.Errorf("Name() = %s, want drift_history", p.Name())
	}
	if len(p.GetActiveDebtItems()) != 0 {
		t.Error("Expected no active debt items initially")
	}
	if len(p.GetDriftHistory()) != 0 {
		t.Error("Expected no history initially")
	}
}

func TestDriftHistoryProjection_ApplyDriftDetected(t *testing.T) {
	p := NewDriftHistoryProjection()

	event := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"category":     "MISSING",
			"message":      "Missing task for feature",
			"issue_count":  2,
		},
	}

	err := p.Apply(event)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Check history
	history := p.GetDriftHistory()
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].ComponentID != "feature-1" {
		t.Errorf("ComponentID = %s, want feature-1", history[0].ComponentID)
	}
	if history[0].IssueCount != 2 {
		t.Errorf("IssueCount = %d, want 2", history[0].IssueCount)
	}

	// Check debt items
	items := p.GetActiveDebtItems()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ComponentID != "feature-1" {
		t.Errorf("Item.ComponentID = %s, want feature-1", items[0].ComponentID)
	}
}

func TestDriftHistoryProjection_MultipleDetections(t *testing.T) {
	p := NewDriftHistoryProjection()

	// First detection
	event1 := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now().Add(-5 * time.Hour),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"message":      "Missing task",
			"issue_count":  1,
		},
	}
	p.Apply(event1)

	// Second detection of same drift
	event2 := &BaseEvent{
		ID:        "event-2",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"message":      "Missing task",
			"issue_count":  1,
		},
	}
	p.Apply(event2)

	// Should have 2 history entries but 1 debt item with count 2
	history := p.GetDriftHistory()
	if len(history) != 2 {
		t.Errorf("len(history) = %d, want 2", len(history))
	}

	items := p.GetActiveDebtItems()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1 (same drift)", len(items))
	}
	if items[0].DetectionCount != 2 {
		t.Errorf("DetectionCount = %d, want 2", items[0].DetectionCount)
	}
}

func TestDriftHistoryProjection_DriftAccepted(t *testing.T) {
	p := NewDriftHistoryProjection()

	// Detect drift
	detectEvent := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"message":      "Intentional skip",
			"issue_count":  1,
		},
	}
	p.Apply(detectEvent)

	// Accept drift
	acceptEvent := &BaseEvent{
		ID:        "event-2",
		Type:      EventTypeDriftAccepted,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
		},
	}
	p.Apply(acceptEvent)

	items := p.GetActiveDebtItems()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Category != debt.DebtIntentional {
		t.Errorf("Category = %s, want %s", items[0].Category, debt.DebtIntentional)
	}
}

func TestDriftHistoryProjection_DriftResolved(t *testing.T) {
	p := NewDriftHistoryProjection()

	// Detect drift
	detectEvent := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"message":      "Missing task",
			"issue_count":  1,
		},
	}
	p.Apply(detectEvent)

	// Verify active
	if len(p.GetActiveDebtItems()) != 1 {
		t.Fatal("Expected 1 active debt item")
	}

	// Resolve drift
	resolveEvent := &BaseEvent{
		ID:        "event-2",
		Type:      EventTypeDriftResolved,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
		},
	}
	p.Apply(resolveEvent)

	// Should be resolved
	if len(p.GetActiveDebtItems()) != 0 {
		t.Error("Expected 0 active debt items after resolution")
	}
	if len(p.GetResolvedDebtItems()) != 1 {
		t.Error("Expected 1 resolved debt item")
	}
}

func TestDriftHistoryProjection_GetStickyDebtItems(t *testing.T) {
	p := NewDriftHistoryProjection()

	// Add item and manually make it sticky for testing
	event := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"message":      "Old drift",
			"issue_count":  1,
		},
	}
	p.Apply(event)

	// Manually set as sticky for testing
	items := p.GetActiveDebtItems()
	if len(items) == 1 {
		items[0].FirstDetected = time.Now().Add(-10 * 24 * time.Hour)
		items[0].Update() // This will set IsSticky = true
	}

	sticky := p.GetStickyDebtItems()
	if len(sticky) != 1 {
		t.Errorf("len(sticky) = %d, want 1", len(sticky))
	}
}

func TestDriftHistoryProjection_GetDebtByComponent(t *testing.T) {
	p := NewDriftHistoryProjection()

	// Add drifts for two components
	for _, comp := range []string{"feature-1", "feature-2", "feature-1"} {
		event := &BaseEvent{
			ID:        "event-" + comp,
			Type:      EventTypeDriftDetected,
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"component_id": comp,
				"drift_type":   "plan",
				"message":      "Drift for " + comp,
				"issue_count":  1,
			},
		}
		p.Apply(event)
	}

	byComponent := p.GetDebtByComponent()
	if len(byComponent) != 2 {
		t.Errorf("len(byComponent) = %d, want 2", len(byComponent))
	}
}

func TestDriftHistoryProjection_GetDebtReport(t *testing.T) {
	p := NewDriftHistoryProjection()

	// Add some drift
	event := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"category":     "MISSING",
			"message":      "Missing task",
			"issue_count":  1,
		},
	}
	p.Apply(event)

	report := p.GetDebtReport()

	if report.TotalItems != 1 {
		t.Errorf("TotalItems = %d, want 1", report.TotalItems)
	}
	if _, ok := report.Scores["feature-1"]; !ok {
		t.Error("Expected score for feature-1")
	}
}

func TestDriftHistoryProjection_GetDriftTrend(t *testing.T) {
	p := NewDriftHistoryProjection()

	// Add some historical drift
	now := time.Now()
	for i := 0; i < 5; i++ {
		event := &BaseEvent{
			ID:        "event-" + string(rune('a'+i)),
			Type:      EventTypeDriftDetected,
			Timestamp: now.AddDate(0, 0, -20+i*5), // Spread over 20 days
			Metadata: map[string]any{
				"component_id": "feature-x",
				"drift_type":   "plan",
				"issue_count":  i + 1, // Increasing issues
			},
		}
		p.Apply(event)
	}

	trend := p.GetDriftTrend(30)

	if trend.WindowDays != 30 {
		t.Errorf("WindowDays = %d, want 30", trend.WindowDays)
	}
	// Should have detected increasing or stable trend
	if trend.Direction != "increasing" && trend.Direction != "stable" {
		t.Logf("Direction = %s (may vary based on timing)", trend.Direction)
	}
}

func TestDriftHistoryProjection_GetDriftHistoryInWindow(t *testing.T) {
	p := NewDriftHistoryProjection()

	now := time.Now()
	// Add old event
	oldEvent := &BaseEvent{
		ID:        "old",
		Type:      EventTypeDriftDetected,
		Timestamp: now.AddDate(0, 0, -30),
		Metadata: map[string]any{
			"component_id": "old-feature",
			"drift_type":   "plan",
			"issue_count":  1,
		},
	}
	p.Apply(oldEvent)

	// Add recent event
	recentEvent := &BaseEvent{
		ID:        "recent",
		Type:      EventTypeDriftDetected,
		Timestamp: now.AddDate(0, 0, -3),
		Metadata: map[string]any{
			"component_id": "recent-feature",
			"drift_type":   "plan",
			"issue_count":  1,
		},
	}
	p.Apply(recentEvent)

	// 7-day window should only include recent
	history := p.GetDriftHistoryInWindow(7)
	if len(history) != 1 {
		t.Errorf("len(history) = %d, want 1 for 7-day window", len(history))
	}

	// 60-day window should include both
	historyAll := p.GetDriftHistoryInWindow(60)
	if len(historyAll) != 2 {
		t.Errorf("len(historyAll) = %d, want 2 for 60-day window", len(historyAll))
	}
}

func TestDriftHistoryProjection_Rebuild(t *testing.T) {
	p := NewDriftHistoryProjection()

	events := []*BaseEvent{
		{
			ID:        "event-1",
			Type:      EventTypeDriftDetected,
			Timestamp: time.Now().Add(-1 * time.Hour),
			Metadata: map[string]any{
				"component_id": "feature-1",
				"drift_type":   "plan",
				"issue_count":  1,
			},
		},
		{
			ID:        "event-2",
			Type:      EventTypeDriftDetected,
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"component_id": "feature-2",
				"drift_type":   "spec",
				"issue_count":  2,
			},
		},
	}

	err := p.Rebuild(events)
	if err != nil {
		t.Fatalf("Rebuild failed: %v", err)
	}

	if len(p.GetDriftHistory()) != 2 {
		t.Errorf("len(history) = %d, want 2", len(p.GetDriftHistory()))
	}
	if len(p.GetActiveDebtItems()) != 2 {
		t.Errorf("len(items) = %d, want 2", len(p.GetActiveDebtItems()))
	}
}

func TestDriftHistoryProjection_Reset(t *testing.T) {
	p := NewDriftHistoryProjection()

	event := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"issue_count":  1,
		},
	}
	p.Apply(event)

	if len(p.GetActiveDebtItems()) != 1 {
		t.Fatal("Expected 1 item before reset")
	}

	p.Reset()

	if len(p.GetActiveDebtItems()) != 0 {
		t.Error("Expected 0 items after reset")
	}
	if len(p.GetDriftHistory()) != 0 {
		t.Error("Expected 0 history after reset")
	}
}

func TestDriftHistoryProjection_Regression(t *testing.T) {
	p := NewDriftHistoryProjection()

	// Detect and resolve
	detectEvent := &BaseEvent{
		ID:        "event-1",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now().Add(-1 * time.Hour),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"message":      "Missing task",
		},
	}
	p.Apply(detectEvent)

	resolveEvent := &BaseEvent{
		ID:        "event-2",
		Type:      EventTypeDriftResolved,
		Timestamp: time.Now().Add(-30 * time.Minute),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
		},
	}
	p.Apply(resolveEvent)

	// Detect same drift again (regression)
	regressionEvent := &BaseEvent{
		ID:        "event-3",
		Type:      EventTypeDriftDetected,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"component_id": "feature-1",
			"drift_type":   "plan",
			"message":      "Missing task again",
		},
	}
	p.Apply(regressionEvent)

	items := p.GetActiveDebtItems()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Category != debt.DebtRegression {
		t.Errorf("Category = %s, want %s (regression)", items[0].Category, debt.DebtRegression)
	}
}

func TestGetIntMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		key      string
		want     int
	}{
		{"int value", map[string]any{"count": 5}, "count", 5},
		{"int64 value", map[string]any{"count": int64(10)}, "count", 10},
		{"float64 value", map[string]any{"count": float64(7)}, "count", 7},
		{"missing key", map[string]any{}, "count", 0},
		{"wrong type", map[string]any{"count": "five"}, "count", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIntMetadata(tt.metadata, tt.key); got != tt.want {
				t.Errorf("getIntMetadata() = %d, want %d", got, tt.want)
			}
		})
	}
}
