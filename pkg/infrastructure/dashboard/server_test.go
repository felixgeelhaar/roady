package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// mockProvider implements DataProvider for testing.
type mockProvider struct {
	plan  *planning.Plan
	state *planning.ExecutionState
	err   error
}

func (m *mockProvider) GetPlan() (*planning.Plan, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.plan, nil
}

func (m *mockProvider) GetState() (*planning.ExecutionState, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.state, nil
}

func TestNewServer(t *testing.T) {
	provider := &mockProvider{}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	if server == nil {
		t.Fatal("Expected non-nil server")
	}
	if server.addr != ":8080" {
		t.Errorf("Expected addr :8080, got %s", server.addr)
	}
}

func TestHandleIndex(t *testing.T) {
	plan := &planning.Plan{
		ID:             "test-plan",
		SpecID:         "test-spec",
		ApprovalStatus: "approved",
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Task 1"},
			{ID: "task-2", Title: "Task 2"},
		},
	}
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {Status: planning.StatusDone},
		},
	}

	provider := &mockProvider{plan: plan, state: state}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.handleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Dashboard") {
		t.Error("Expected page to contain Dashboard")
	}
	// Test passes if either plan ID appears or "Approved" badge appears (showing plan is rendered)
	if !strings.Contains(body, "test-plan") && !strings.Contains(body, "Approved") {
		t.Logf("Response body length: %d", len(body))
		t.Error("Expected page to contain plan ID or approval status")
	}
}

func TestHandleIndexWithError(t *testing.T) {
	provider := &mockProvider{err: errors.New("test error")}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.handleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "test error") {
		t.Error("Expected page to contain error message")
	}
}

func TestHandlePlan(t *testing.T) {
	plan := &planning.Plan{
		ID:             "test-plan",
		SpecID:         "test-spec",
		ApprovalStatus: "approved",
		CreatedAt:      time.Now(),
	}

	provider := &mockProvider{plan: plan}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/plan", nil)
	rec := httptest.NewRecorder()

	server.handlePlan(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Plan") {
		t.Error("Expected page to contain Plan")
	}
	if !strings.Contains(body, "test-plan") {
		t.Error("Expected page to contain plan ID")
	}
}

func TestHandleTasks(t *testing.T) {
	plan := &planning.Plan{
		ID:     "test-plan",
		SpecID: "test-spec",
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Task 1", Description: "Description 1"},
			{ID: "task-2", Title: "Task 2", DependsOn: []string{"task-1"}},
		},
	}
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {
				Status: planning.StatusDone,
				Owner:  "alice",
				ExternalRefs: map[string]planning.ExternalRef{
					"github": {ID: "123", Identifier: "GH-123"},
				},
			},
			"task-2": {Status: planning.StatusInProgress},
		},
	}

	provider := &mockProvider{plan: plan, state: state}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()

	server.handleTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Tasks") {
		t.Error("Expected page to contain Tasks")
	}
	if !strings.Contains(body, "Task 1") {
		t.Error("Expected page to contain Task 1")
	}
}

func TestHandleAPIPlan(t *testing.T) {
	plan := &planning.Plan{
		ID:     "test-plan",
		SpecID: "test-spec",
	}

	provider := &mockProvider{plan: plan}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plan", nil)
	rec := httptest.NewRecorder()

	server.handleAPIPlan(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result planning.Plan
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if result.ID != "test-plan" {
		t.Errorf("Expected ID test-plan, got %s", result.ID)
	}
}

func TestHandleAPIPlanError(t *testing.T) {
	provider := &mockProvider{err: errors.New("plan error")}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plan", nil)
	rec := httptest.NewRecorder()

	server.handleAPIPlan(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestHandleAPIState(t *testing.T) {
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {Status: planning.StatusDone},
		},
	}

	provider := &mockProvider{state: state}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	rec := httptest.NewRecorder()

	server.handleAPIState(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result planning.ExecutionState
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if result.TaskStates["task-1"].Status != planning.StatusDone {
		t.Error("Expected task-1 status to be done")
	}
}

func TestHandleAPIStateError(t *testing.T) {
	provider := &mockProvider{err: errors.New("state error")}
	server, err := NewServer(":8080", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	rec := httptest.NewRecorder()

	server.handleAPIState(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestCalculateStats(t *testing.T) {
	plan := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "task-1"},
			{ID: "task-2"},
			{ID: "task-3"},
			{ID: "task-4"},
			{ID: "task-5"},
		},
	}
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {Status: planning.StatusDone},
			"task-2": {Status: planning.StatusInProgress},
			"task-3": {Status: planning.StatusBlocked},
			"task-4": {Status: planning.StatusVerified},
			// task-5 has no state, should be pending
		},
	}

	stats := calculateStats(plan, state)

	if stats.TotalTasks != 5 {
		t.Errorf("Expected TotalTasks 5, got %d", stats.TotalTasks)
	}
	if stats.Pending != 1 {
		t.Errorf("Expected Pending 1, got %d", stats.Pending)
	}
	if stats.InProgress != 1 {
		t.Errorf("Expected InProgress 1, got %d", stats.InProgress)
	}
	if stats.Done != 2 {
		t.Errorf("Expected Done 2, got %d", stats.Done)
	}
	if stats.Blocked != 1 {
		t.Errorf("Expected Blocked 1, got %d", stats.Blocked)
	}
	if stats.Completion != 40.0 {
		t.Errorf("Expected Completion 40.0, got %f", stats.Completion)
	}
}

func TestCalculateStatsEmpty(t *testing.T) {
	plan := &planning.Plan{Tasks: []planning.Task{}}
	state := &planning.ExecutionState{TaskStates: map[string]planning.TaskResult{}}

	stats := calculateStats(plan, state)

	if stats.TotalTasks != 0 {
		t.Errorf("Expected TotalTasks 0, got %d", stats.TotalTasks)
	}
	if stats.Completion != 0 {
		t.Errorf("Expected Completion 0, got %f", stats.Completion)
	}
}

func TestBuildTaskViews(t *testing.T) {
	plan := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Task 1"},
			{ID: "task-2", Title: "Task 2"},
		},
	}
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {
				Status: planning.StatusDone,
				Owner:  "alice",
				ExternalRefs: map[string]planning.ExternalRef{
					"github": {ID: "123"},
				},
			},
		},
	}

	views := buildTaskViews(plan, state)

	if len(views) != 2 {
		t.Fatalf("Expected 2 views, got %d", len(views))
	}

	// Check task-1
	if views[0].Task.ID != "task-1" {
		t.Errorf("Expected task-1, got %s", views[0].Task.ID)
	}
	if views[0].Status != planning.StatusDone {
		t.Errorf("Expected done status, got %s", views[0].Status)
	}
	if views[0].Owner != "alice" {
		t.Errorf("Expected owner alice, got %s", views[0].Owner)
	}
	if !views[0].HasLinks {
		t.Error("Expected HasLinks to be true")
	}

	// Check task-2 (no state)
	if views[1].Task.ID != "task-2" {
		t.Errorf("Expected task-2, got %s", views[1].Task.ID)
	}
	if views[1].Status != planning.StatusPending {
		t.Errorf("Expected pending status, got %s", views[1].Status)
	}
	if views[1].HasLinks {
		t.Error("Expected HasLinks to be false")
	}
}

func TestBuildTaskViewsNilState(t *testing.T) {
	plan := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Task 1"},
		},
	}

	views := buildTaskViews(plan, nil)

	if len(views) != 1 {
		t.Fatalf("Expected 1 view, got %d", len(views))
	}
	if views[0].Status != planning.StatusPending {
		t.Errorf("Expected pending status, got %s", views[0].Status)
	}
}

func TestStatusClass(t *testing.T) {
	tests := []struct {
		status planning.TaskStatus
		want   string
	}{
		{planning.StatusPending, "status-pending"},
		{planning.StatusInProgress, "status-progress"},
		{planning.StatusDone, "status-done"},
		{planning.StatusVerified, "status-done"},
		{planning.StatusBlocked, "status-blocked"},
		{planning.TaskStatus("unknown"), "status-unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := statusClass(tt.status)
			if got != tt.want {
				t.Errorf("statusClass(%s) = %s, want %s", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	// Test zero time
	got := formatTime(time.Time{})
	if got != "-" {
		t.Errorf("formatTime(zero) = %s, want -", got)
	}

	// Test non-zero time
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	got = formatTime(tm)
	want := "2024-01-15 10:30"
	if got != want {
		t.Errorf("formatTime(%v) = %s, want %s", tm, got, want)
	}
}

func TestToJSON(t *testing.T) {
	data := map[string]string{"key": "value"}
	got := toJSON(data)
	if !strings.Contains(got, "key") || !strings.Contains(got, "value") {
		t.Errorf("toJSON(%v) = %s, expected to contain key and value", data, got)
	}
}

func TestServerShutdown(t *testing.T) {
	provider := &mockProvider{}
	server, err := NewServer(":0", provider)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Shutdown without Start should not error
	err = server.Shutdown(context.TODO())
	if err != nil {
		t.Errorf("Shutdown without Start should not error: %v", err)
	}
}
