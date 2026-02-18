package application

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

// mockForecastRepo implements domain.WorkspaceRepository for testing
type mockForecastRepo struct {
	plan  *planning.Plan
	state *planning.ExecutionState
}

func (m *mockForecastRepo) Initialize() error                          { return nil }
func (m *mockForecastRepo) IsInitialized() bool                        { return true }
func (m *mockForecastRepo) SaveSpec(s *spec.ProductSpec) error         { return nil }
func (m *mockForecastRepo) LoadSpec() (*spec.ProductSpec, error)       { return nil, nil }
func (m *mockForecastRepo) SaveSpecLock(s *spec.ProductSpec) error     { return nil }
func (m *mockForecastRepo) LoadSpecLock() (*spec.ProductSpec, error)   { return nil, nil }
func (m *mockForecastRepo) SavePlan(p *planning.Plan) error            { m.plan = p; return nil }
func (m *mockForecastRepo) LoadPlan() (*planning.Plan, error)          { return m.plan, nil }
func (m *mockForecastRepo) SaveState(s *planning.ExecutionState) error { m.state = s; return nil }
func (m *mockForecastRepo) LoadState() (*planning.ExecutionState, error) {
	return m.state, nil
}
func (m *mockForecastRepo) RecordEvent(e domain.Event) error          { return nil }
func (m *mockForecastRepo) LoadEvents() ([]domain.Event, error)       { return nil, nil }
func (m *mockForecastRepo) UpdateUsage(s domain.UsageStats) error     { return nil }
func (m *mockForecastRepo) LoadUsage() (*domain.UsageStats, error)    { return nil, nil }
func (m *mockForecastRepo) SavePolicy(c *domain.PolicyConfig) error   { return nil }
func (m *mockForecastRepo) LoadPolicy() (*domain.PolicyConfig, error) { return nil, nil }
func (m *mockForecastRepo) SaveRates(c *billing.RateConfig) error     { return nil }
func (m *mockForecastRepo) LoadRates() (*billing.RateConfig, error) {
	return &billing.RateConfig{}, nil
}
func (m *mockForecastRepo) SaveTimeEntries(e []billing.TimeEntry) error { return nil }
func (m *mockForecastRepo) LoadTimeEntries() ([]billing.TimeEntry, error) {
	return []billing.TimeEntry{}, nil
}

func TestForecastService_GetForecast_NoPlan(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()
	repo := &mockForecastRepo{plan: nil, state: nil}

	svc := NewForecastService(projection, repo)

	forecast, err := svc.GetForecast()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if forecast != nil {
		t.Error("Expected nil forecast when no plan exists")
	}
}

func TestForecastService_GetForecast_WithPlan(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()

	// Add some completion events
	now := time.Now()
	for i := 0; i < 5; i++ {
		_ = projection.Apply(&events.BaseEvent{
			Type:      events.EventTypeTaskCompleted,
			Timestamp: now.AddDate(0, 0, -i),
			Metadata:  map[string]interface{}{"task_id": "completed-task"},
		})
	}

	plan := &planning.Plan{
		ID:     "plan-1",
		SpecID: "spec-1",
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Task 1"},
			{ID: "task-2", Title: "Task 2"},
			{ID: "task-3", Title: "Task 3"},
			{ID: "task-4", Title: "Task 4"},
			{ID: "task-5", Title: "Task 5"},
		},
	}

	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {Status: planning.StatusVerified},
			"task-2": {Status: planning.StatusDone},
		},
	}

	repo := &mockForecastRepo{plan: plan, state: state}
	svc := NewForecastService(projection, repo)

	forecast, err := svc.GetForecast()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if forecast == nil {
		t.Fatal("Expected forecast to be non-nil")
	}

	if forecast.TotalTasks != 5 {
		t.Errorf("Expected 5 total tasks, got %d", forecast.TotalTasks)
	}
	if forecast.CompletedTasks != 2 {
		t.Errorf("Expected 2 completed tasks, got %d", forecast.CompletedTasks)
	}
	if forecast.RemainingTasks != 3 {
		t.Errorf("Expected 3 remaining tasks, got %d", forecast.RemainingTasks)
	}
	if forecast.Velocity <= 0 {
		t.Error("Expected positive velocity")
	}
}

func TestForecastService_GetForecast_NoVelocity(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()
	// No completion events = no velocity

	plan := &planning.Plan{
		ID:    "plan-1",
		Tasks: []planning.Task{{ID: "task-1"}},
	}

	repo := &mockForecastRepo{plan: plan, state: nil}
	svc := NewForecastService(projection, repo)

	forecast, err := svc.GetForecast()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if forecast.Velocity != 0 {
		t.Errorf("Expected 0 velocity with no data, got %f", forecast.Velocity)
	}
	if forecast.EstimatedDays != 0 {
		t.Errorf("Expected 0 estimated days with no velocity, got %f", forecast.EstimatedDays)
	}
}

func TestForecastService_GetVelocityTrend(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()
	repo := &mockForecastRepo{plan: nil, state: nil}
	svc := NewForecastService(projection, repo)

	trend := svc.GetVelocityTrend()

	// With no data, should be stable
	if trend.Direction != analytics.TrendStable {
		t.Errorf("Expected stable trend with no data, got %s", trend.Direction)
	}
}

func TestForecastService_GetVelocityStats(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()
	repo := &mockForecastRepo{plan: nil, state: nil}
	svc := NewForecastService(projection, repo)

	stats := svc.GetVelocityStats()

	if stats.Samples != 0 {
		t.Errorf("Expected 0 samples with no data, got %d", stats.Samples)
	}
}

func TestForecastService_GetBurndown_NoPlan(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()
	repo := &mockForecastRepo{plan: nil, state: nil}
	svc := NewForecastService(projection, repo)

	burndown, err := svc.GetBurndown()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if burndown != nil {
		t.Error("Expected nil burndown when no plan exists")
	}
}

func TestForecastService_GetVelocityWindows(t *testing.T) {
	projection := events.NewExtendedVelocityProjection(7, 14)
	repo := &mockForecastRepo{plan: nil, state: nil}
	svc := NewForecastService(projection, repo)

	windows := svc.GetVelocityWindows()

	if len(windows) != 2 {
		t.Errorf("Expected 2 windows, got %d", len(windows))
	}
}

func TestForecastService_GetSimpleForecast(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()

	now := time.Now()
	for i := 0; i < 3; i++ {
		_ = projection.Apply(&events.BaseEvent{
			Type:      events.EventTypeTaskCompleted,
			Timestamp: now.AddDate(0, 0, -i),
			Metadata:  map[string]interface{}{"task_id": "t"},
		})
	}

	plan := &planning.Plan{
		ID:    "plan-1",
		Tasks: []planning.Task{{ID: "task-1"}, {ID: "task-2"}},
	}

	repo := &mockForecastRepo{plan: plan, state: nil}
	svc := NewForecastService(projection, repo)

	simple, err := svc.GetSimpleForecast()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if simple == nil {
		t.Fatal("Expected non-nil simple forecast")
	}

	if simple.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", simple.TotalTasks)
	}
	if simple.RemainingTasks != 2 {
		t.Errorf("Expected 2 remaining tasks, got %d", simple.RemainingTasks)
	}
}

func TestForecastService_GetSimpleForecast_NoPlan(t *testing.T) {
	projection := events.NewExtendedVelocityProjection()
	repo := &mockForecastRepo{plan: nil, state: nil}
	svc := NewForecastService(projection, repo)

	simple, err := svc.GetSimpleForecast()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if simple != nil {
		t.Error("Expected nil simple forecast when no plan exists")
	}
}

func TestForecastService_ConfidenceInterval_Accelerating(t *testing.T) {
	svc := &ForecastService{}

	trend := analytics.VelocityTrend{
		Direction:  analytics.TrendAccelerating,
		Confidence: 0.8,
	}

	ci := svc.calculateConfidenceInterval(10, 2.0, trend)

	// Expected = 10/2 = 5 days
	// With acceleration: low = 5*0.7 = 3.5, high = 5*1.2 = 6
	if ci.Expected != 5.0 {
		t.Errorf("Expected 5.0 days, got %f", ci.Expected)
	}
	if ci.Low >= ci.Expected {
		t.Error("Low estimate should be less than expected for acceleration")
	}
	if ci.High <= ci.Expected {
		t.Error("High estimate should be greater than expected")
	}
}

func TestForecastService_ConfidenceInterval_Decelerating(t *testing.T) {
	svc := &ForecastService{}

	trend := analytics.VelocityTrend{
		Direction:  analytics.TrendDecelerating,
		Confidence: 0.8,
	}

	ci := svc.calculateConfidenceInterval(10, 2.0, trend)

	// With deceleration, high should be much higher
	if ci.High <= ci.Expected*1.5 {
		t.Error("High estimate should be significantly greater for deceleration")
	}
}

func TestForecastService_ConfidenceInterval_LowConfidence(t *testing.T) {
	svc := &ForecastService{}

	lowConfTrend := analytics.VelocityTrend{
		Direction:  analytics.TrendStable,
		Confidence: 0.3,
	}

	highConfTrend := analytics.VelocityTrend{
		Direction:  analytics.TrendStable,
		Confidence: 0.9,
	}

	ciLow := svc.calculateConfidenceInterval(10, 2.0, lowConfTrend)
	ciHigh := svc.calculateConfidenceInterval(10, 2.0, highConfTrend)

	// Low confidence should have wider interval
	lowRange := ciLow.High - ciLow.Low
	highRange := ciHigh.High - ciHigh.Low

	if lowRange <= highRange {
		t.Error("Low confidence should result in wider interval")
	}
}

func TestForecastService_ConfidenceInterval_ZeroVelocity(t *testing.T) {
	svc := &ForecastService{}

	trend := analytics.VelocityTrend{Direction: analytics.TrendStable}

	ci := svc.calculateConfidenceInterval(10, 0, trend)

	if ci.Expected != 0 || ci.Low != 0 || ci.High != 0 {
		t.Error("Zero velocity should result in zero confidence interval")
	}
}

func TestForecastService_ConfidenceInterval_ZeroRemaining(t *testing.T) {
	svc := &ForecastService{}

	trend := analytics.VelocityTrend{Direction: analytics.TrendStable}

	ci := svc.calculateConfidenceInterval(0, 2.0, trend)

	if ci.Expected != 0 || ci.Low != 0 || ci.High != 0 {
		t.Error("Zero remaining should result in zero confidence interval")
	}
}
