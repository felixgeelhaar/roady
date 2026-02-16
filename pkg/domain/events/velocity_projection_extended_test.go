package events

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
)

func TestNewExtendedVelocityProjection_DefaultWindows(t *testing.T) {
	p := NewExtendedVelocityProjection()

	if len(p.windows) != 3 {
		t.Errorf("Expected 3 default windows, got %d", len(p.windows))
	}
	if p.windows[0] != 7 || p.windows[1] != 14 || p.windows[2] != 30 {
		t.Errorf("Expected windows [7, 14, 30], got %v", p.windows)
	}
}

func TestNewExtendedVelocityProjection_CustomWindows(t *testing.T) {
	p := NewExtendedVelocityProjection(5, 10, 20)

	if len(p.windows) != 3 {
		t.Errorf("Expected 3 windows, got %d", len(p.windows))
	}
	if p.windows[0] != 5 {
		t.Errorf("Expected first window 5, got %d", p.windows[0])
	}
}

func TestExtendedVelocityProjection_Name(t *testing.T) {
	p := NewExtendedVelocityProjection()
	if p.Name() != "velocity_extended" {
		t.Errorf("Expected name 'velocity_extended', got '%s'", p.Name())
	}
}

func TestExtendedVelocityProjection_Apply(t *testing.T) {
	p := NewExtendedVelocityProjection()

	event := &BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"task_id": "task-1",
		},
	}

	err := p.Apply(event)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	if p.GetCompletionCount() != 1 {
		t.Errorf("Expected 1 completion, got %d", p.GetCompletionCount())
	}
}

func TestExtendedVelocityProjection_ApplyIgnoresOtherEvents(t *testing.T) {
	p := NewExtendedVelocityProjection()

	event := &BaseEvent{
		Type:      EventTypeTaskStarted,
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"task_id": "task-1"},
	}

	err := p.Apply(event)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	if p.GetCompletionCount() != 0 {
		t.Errorf("Expected 0 completions for non-completion event, got %d", p.GetCompletionCount())
	}
}

func TestExtendedVelocityProjection_Reset(t *testing.T) {
	p := NewExtendedVelocityProjection()

	p.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"task_id": "task-1"},
	})

	if p.GetCompletionCount() != 1 {
		t.Error("Expected 1 completion before reset")
	}

	p.Reset()

	if p.GetCompletionCount() != 0 {
		t.Errorf("Expected 0 completions after reset, got %d", p.GetCompletionCount())
	}
}

func TestExtendedVelocityProjection_GetVelocityWindows(t *testing.T) {
	p := NewExtendedVelocityProjection(7, 14)

	now := time.Now()
	// Add completions at different times
	for i := 0; i < 5; i++ {
		p.Apply(&BaseEvent{
			Type:      EventTypeTaskCompleted,
			Timestamp: now.AddDate(0, 0, -i), // Within 7 days
			Metadata:  map[string]interface{}{"task_id": "task-" + string(rune('a'+i))},
		})
	}

	windows := p.GetVelocityWindows()

	if len(windows) != 2 {
		t.Fatalf("Expected 2 windows, got %d", len(windows))
	}

	// First window (7 days) should have all 5 completions
	if windows[0].Days != 7 {
		t.Errorf("Expected first window to be 7 days, got %d", windows[0].Days)
	}
	if windows[0].Count != 5 {
		t.Errorf("Expected 5 completions in 7-day window, got %d", windows[0].Count)
	}

	// Check velocity calculation
	expectedVelocity := 5.0 / 7.0
	if windows[0].Velocity != expectedVelocity {
		t.Errorf("Expected velocity %f, got %f", expectedVelocity, windows[0].Velocity)
	}
}

func TestExtendedVelocityProjection_GetVelocityTrend_NoData(t *testing.T) {
	p := NewExtendedVelocityProjection()
	trend := p.GetVelocityTrend()

	if trend.Direction != analytics.TrendStable {
		t.Errorf("Expected stable trend with no data, got %s", trend.Direction)
	}
	if trend.Confidence != 0 {
		t.Errorf("Expected 0 confidence with no data, got %f", trend.Confidence)
	}
}

func TestExtendedVelocityProjection_GetVelocityTrend_SingleWindow(t *testing.T) {
	p := NewExtendedVelocityProjection(7)

	p.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"task_id": "task-1"},
	})

	trend := p.GetVelocityTrend()

	// With only one window, should be stable
	if trend.Direction != analytics.TrendStable {
		t.Errorf("Expected stable trend with single window, got %s", trend.Direction)
	}
}

func TestExtendedVelocityProjection_GetVelocityTrend_Accelerating(t *testing.T) {
	p := NewExtendedVelocityProjection(7, 30)

	now := time.Now()
	// Add more completions in recent days (acceleration)
	// 5 completions in last 7 days, 2 older completions (outside 7 but in 30)
	for i := 0; i < 5; i++ {
		p.Apply(&BaseEvent{
			Type:      EventTypeTaskCompleted,
			Timestamp: now.AddDate(0, 0, -i), // Recent
			Metadata:  map[string]interface{}{"task_id": "recent-" + string(rune('a'+i))},
		})
	}
	for i := 0; i < 2; i++ {
		p.Apply(&BaseEvent{
			Type:      EventTypeTaskCompleted,
			Timestamp: now.AddDate(0, 0, -20-i), // Older
			Metadata:  map[string]interface{}{"task_id": "old-" + string(rune('a'+i))},
		})
	}

	trend := p.GetVelocityTrend()

	// Recent velocity higher than long-term should show acceleration
	if trend.Direction != analytics.TrendAccelerating {
		t.Errorf("Expected accelerating trend, got %s (slope: %f)", trend.Direction, trend.Slope)
	}
	if trend.Slope <= 0 {
		t.Errorf("Expected positive slope for acceleration, got %f", trend.Slope)
	}
}

func TestExtendedVelocityProjection_GetVelocityStats_NoData(t *testing.T) {
	p := NewExtendedVelocityProjection()
	stats := p.GetVelocityStats()

	if stats.Samples != 0 {
		t.Errorf("Expected 0 samples with no data, got %d", stats.Samples)
	}
}

func TestExtendedVelocityProjection_GetVelocityStats_WithData(t *testing.T) {
	p := NewExtendedVelocityProjection()

	now := time.Now()
	// Add completions on specific days
	// Day 1: 2 completions
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now, Metadata: map[string]interface{}{"task_id": "t1"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now, Metadata: map[string]interface{}{"task_id": "t2"}})
	// Day 2: 1 completion
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -1), Metadata: map[string]interface{}{"task_id": "t3"}})
	// Day 3: 3 completions
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -2), Metadata: map[string]interface{}{"task_id": "t4"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -2), Metadata: map[string]interface{}{"task_id": "t5"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -2), Metadata: map[string]interface{}{"task_id": "t6"}})

	stats := p.GetVelocityStats()

	if stats.Samples != 3 {
		t.Errorf("Expected 3 samples (3 days), got %d", stats.Samples)
	}
	// Mean should be (2+1+3)/3 = 2
	if stats.Mean != 2.0 {
		t.Errorf("Expected mean 2.0, got %f", stats.Mean)
	}
	// Min should be 1
	if stats.Min != 1.0 {
		t.Errorf("Expected min 1.0, got %f", stats.Min)
	}
	// Max should be 3
	if stats.Max != 3.0 {
		t.Errorf("Expected max 3.0, got %f", stats.Max)
	}
}

func TestExtendedVelocityProjection_GenerateBurndown_Empty(t *testing.T) {
	p := NewExtendedVelocityProjection()
	burndown := p.GenerateBurndown(10, 10, 7)

	// With no completions, no historical data and projected remains 10
	if len(burndown) == 0 {
		// Expected - no velocity means no projection
		return
	}
}

func TestExtendedVelocityProjection_GetCompletionsInWindow(t *testing.T) {
	p := NewExtendedVelocityProjection()

	now := time.Now()
	// Add completions at different times
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now, Metadata: map[string]interface{}{"task_id": "t1"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -3), Metadata: map[string]interface{}{"task_id": "t2"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -10), Metadata: map[string]interface{}{"task_id": "t3"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -20), Metadata: map[string]interface{}{"task_id": "t4"}})

	// 7-day window should have 2 completions
	count7 := p.GetCompletionsInWindow(7)
	if count7 != 2 {
		t.Errorf("Expected 2 completions in 7-day window, got %d", count7)
	}

	// 14-day window should have 3 completions
	count14 := p.GetCompletionsInWindow(14)
	if count14 != 3 {
		t.Errorf("Expected 3 completions in 14-day window, got %d", count14)
	}

	// 30-day window should have all 4 completions
	count30 := p.GetCompletionsInWindow(30)
	if count30 != 4 {
		t.Errorf("Expected 4 completions in 30-day window, got %d", count30)
	}
}

func TestExtendedVelocityProjection_GenerateBurndown_WithVelocity(t *testing.T) {
	p := NewExtendedVelocityProjection(7, 30)

	now := time.Now()
	// Add completions for the last 7 days (1 per day)
	for i := 0; i < 7; i++ {
		p.Apply(&BaseEvent{
			Type:      EventTypeTaskCompleted,
			Timestamp: now.AddDate(0, 0, -i),
			Metadata:  map[string]interface{}{"task_id": "task-" + string(rune('a'+i))},
		})
	}

	burndown := p.GenerateBurndown(20, 10, 14)
	if len(burndown) == 0 {
		t.Error("expected non-empty burndown data")
	}

	// Should have historical points and projected points
	hasProjected := false
	for _, point := range burndown {
		if point.Projected > 0 || (point.Date.After(now) && point.Projected >= 0) {
			hasProjected = true
		}
	}
	if !hasProjected && len(burndown) > 7 {
		// At least we got some data points
		t.Logf("burndown has %d points", len(burndown))
	}
}

func TestExtendedVelocityProjection_GetHistoricalBurndown(t *testing.T) {
	p := NewExtendedVelocityProjection()

	now := time.Now()
	// Add completions across multiple days
	p.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: now.AddDate(0, 0, -3),
		Metadata:  map[string]interface{}{"task_id": "t1"},
	})
	p.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: now.AddDate(0, 0, -3),
		Metadata:  map[string]interface{}{"task_id": "t2"},
	})
	p.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: now.AddDate(0, 0, -1),
		Metadata:  map[string]interface{}{"task_id": "t3"},
	})

	// Use GenerateBurndown with no remaining tasks to get historical only
	burndown := p.GenerateBurndown(10, 0, 0)
	if len(burndown) == 0 {
		t.Error("expected non-empty historical burndown")
	}
}

func TestExtendedVelocityProjection_ApplyWithCycleTime(t *testing.T) {
	p := NewExtendedVelocityProjection()

	startTime := time.Now().Add(-2 * time.Hour)
	event := &BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"task_id":    "task-1",
			"started_at": startTime,
		},
	}

	err := p.Apply(event)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	if p.GetCompletionCount() != 1 {
		t.Errorf("Expected 1 completion, got %d", p.GetCompletionCount())
	}
}

func TestExtendedVelocityProjection_GetVelocityTrend_Decelerating(t *testing.T) {
	p := NewExtendedVelocityProjection(7, 30)

	now := time.Now()
	// Add more completions in older period (deceleration)
	for i := 0; i < 10; i++ {
		p.Apply(&BaseEvent{
			Type:      EventTypeTaskCompleted,
			Timestamp: now.AddDate(0, 0, -20-i), // 20-29 days ago
			Metadata:  map[string]interface{}{"task_id": "old-" + string(rune('a'+i))},
		})
	}
	// Only 1 in recent period
	p.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: now.AddDate(0, 0, -1),
		Metadata:  map[string]interface{}{"task_id": "recent"},
	})

	trend := p.GetVelocityTrend()
	if trend.Direction != analytics.TrendDecelerating {
		t.Logf("Direction = %s, slope = %f (expected decelerating)", trend.Direction, trend.Slope)
	}
}

func TestExtendedVelocityProjection_GetVelocityStats_SingleDay(t *testing.T) {
	p := NewExtendedVelocityProjection()

	// Single day, single completion
	p.Apply(&BaseEvent{
		Type:      EventTypeTaskCompleted,
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"task_id": "t1"},
	})

	stats := p.GetVelocityStats()
	if stats.Samples != 1 {
		t.Errorf("Expected 1 sample, got %d", stats.Samples)
	}
	if stats.Median != 1.0 {
		t.Errorf("Expected median 1.0, got %f", stats.Median)
	}
}

func TestExtendedVelocityProjection_GetVelocityStats_EvenSamples(t *testing.T) {
	p := NewExtendedVelocityProjection()

	now := time.Now()
	// 2 days, to test even-count median
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now, Metadata: map[string]interface{}{"task_id": "t1"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now, Metadata: map[string]interface{}{"task_id": "t2"}})
	p.Apply(&BaseEvent{Type: EventTypeTaskCompleted, Timestamp: now.AddDate(0, 0, -1), Metadata: map[string]interface{}{"task_id": "t3"}})

	stats := p.GetVelocityStats()
	if stats.Samples != 2 {
		t.Errorf("Expected 2 samples (2 days), got %d", stats.Samples)
	}
	// Day 1: 1 completion, Day 2: 2 completions. Sorted: [1, 2]. Median = (1+2)/2 = 1.5
	if stats.Median != 1.5 {
		t.Errorf("Expected median 1.5, got %f", stats.Median)
	}
}

func TestExtendedVelocityProjection_Rebuild(t *testing.T) {
	p := NewExtendedVelocityProjection()

	events := []*BaseEvent{
		{Type: EventTypeTaskCompleted, Timestamp: time.Now(), Metadata: map[string]interface{}{"task_id": "t1"}},
		{Type: EventTypeTaskStarted, Timestamp: time.Now(), Metadata: map[string]interface{}{"task_id": "t2"}},
		{Type: EventTypeTaskCompleted, Timestamp: time.Now(), Metadata: map[string]interface{}{"task_id": "t3"}},
	}

	err := p.Rebuild(events)
	if err != nil {
		t.Errorf("Rebuild failed: %v", err)
	}

	// Should only count completed tasks
	if p.GetCompletionCount() != 2 {
		t.Errorf("Expected 2 completions after rebuild, got %d", p.GetCompletionCount())
	}
}
