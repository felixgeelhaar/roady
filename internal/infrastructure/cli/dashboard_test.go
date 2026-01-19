package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestIsValidBrowserURL(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		valid bool
	}{
		{"http", "http://localhost:3000", true},
		{"https", "https://example.com/path", true},
		{"bad scheme", "file:///etc/passwd", false},
		{"shell char", "http://example.com;rm -rf /", false},
		{"newline", "http://example.com/\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidBrowserURL(tt.url); got != tt.valid {
				t.Fatalf("isValidBrowserURL(%q) = %v, want %v", tt.url, got, tt.valid)
			}
		})
	}
}

func TestOpenBrowser_InvalidURL(t *testing.T) {
	if err := openBrowser("ftp://example.com"); err == nil {
		t.Fatal("expected error for invalid url")
	}
}

func TestInitialModel_Success(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	specDoc := &spec.ProductSpec{
		ID:      "spec-1",
		Title:   "Roady",
		Version: "1.2.3",
		Features: []spec.Feature{
			{ID: "f1", Title: "Feature 1"},
		},
	}
	if err := repo.SaveSpec(specDoc); err != nil {
		t.Fatalf("save spec: %v", err)
	}
	if err := repo.SaveSpecLock(specDoc); err != nil {
		t.Fatalf("save spec lock: %v", err)
	}

	plan := &planning.Plan{
		ID:             "plan-1",
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Task 1", Priority: planning.PriorityHigh},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState("plan-1")
	state.TaskStates["task-1"] = planning.TaskResult{Status: planning.StatusPending, Owner: "me"}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := repo.UpdateUsage(domain.UsageStats{ProviderStats: map[string]int{"openai": 3}}); err != nil {
		t.Fatalf("save usage: %v", err)
	}
	if err := repo.SavePolicy(&domain.PolicyConfig{TokenLimit: 10}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	m := initialModel()
	if m.err != nil {
		t.Fatalf("initialModel returned error: %v", m.err)
	}
	if m.project != "Roady" || m.version != "1.2.3" {
		t.Fatalf("unexpected project/version: %s/%s", m.project, m.version)
	}
	if m.usage != 3 || m.limit != 10 {
		t.Fatalf("unexpected usage/limit: %d/%d", m.usage, m.limit)
	}
}

func TestDashboardModel_ViewAndUpdate(t *testing.T) {
	tbl := table.New(
		table.WithColumns([]table.Column{{Title: "Task", Width: 10}}),
		table.WithRows([]table.Row{{"task"}}),
	)

	m := model{
		table:   tbl,
		project: "Roady",
		version: "1.0.0",
		usage:   2,
		limit:   5,
		drift:   []string{"[medium] drift"},
	}

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := updated.(model); !ok {
		t.Fatalf("expected model update type, got %T", updated)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if _, ok := updated.(model); !ok {
		t.Fatalf("expected model update type, got %T", updated)
	}
}

func TestDashboardModel_ViewError(t *testing.T) {
	m := model{err: errors.New("boom")}
	view := m.View()
	if !strings.Contains(view, "Error loading dashboard") {
		t.Fatalf("expected error view, got:\n%s", view)
	}
}

func TestDashboardDataProvider_GetPlanAndState(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	plan := &planning.Plan{
		ID:             "plan-1",
		ApprovalStatus: planning.ApprovalApproved,
		Tasks:          []planning.Task{{ID: "task-1", Title: "Task 1"}},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState("plan-1")
	state.TaskStates["task-1"] = planning.TaskResult{Status: planning.StatusPending}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	planSvc := application.NewPlanService(repo, application.NewAuditService(repo))
	provider := dashboardDataProvider{services: &wiring.AppServices{Plan: planSvc}}

	gotPlan, err := provider.GetPlan()
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if gotPlan == nil || gotPlan.ID != "plan-1" {
		t.Fatalf("unexpected plan: %#v", gotPlan)
	}

	gotState, err := provider.GetState()
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if gotState == nil || gotState.TaskStates["task-1"].Status != planning.StatusPending {
		t.Fatalf("unexpected state: %#v", gotState)
	}
}

func TestDashboardModel_Init(t *testing.T) {
	m := model{}
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got %v", cmd)
	}
}
