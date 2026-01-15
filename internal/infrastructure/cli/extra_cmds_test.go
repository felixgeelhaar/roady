package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestForecastCmd_NoPlan(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	if err := forecastCmd.RunE(forecastCmd, []string{}); err == nil {
		t.Fatal("expected error when no plan is available")
	}
}

func TestForecastCmd_NoVelocity(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	plan := &planning.Plan{
		ID:             "p1",
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState("p1")
	state.TaskStates["t1"] = planning.TaskResult{Status: planning.StatusPending}
	state.TaskStates["t2"] = planning.TaskResult{Status: planning.StatusPending}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	output := captureStdout(t, func() {
		if err := forecastCmd.RunE(forecastCmd, []string{}); err != nil {
			t.Fatalf("forecast failed: %v", err)
		}
	})

	if !strings.Contains(output, "Unable to forecast") {
		t.Fatalf("expected no-velocity message, got:\n%s", output)
	}
}

func TestForecastCmd_Estimate(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	plan := &planning.Plan{
		ID:             "p1",
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState("p1")
	state.TaskStates["t1"] = planning.TaskResult{Status: planning.StatusVerified}
	state.TaskStates["t2"] = planning.TaskResult{Status: planning.StatusPending}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	audit := application.NewAuditService(repo)
	if err := audit.Log("task.transition", "tester", map[string]interface{}{"status": "verified"}); err != nil {
		t.Fatalf("log event: %v", err)
	}

	output := captureStdout(t, func() {
		if err := forecastCmd.RunE(forecastCmd, []string{}); err != nil {
			t.Fatalf("forecast failed: %v", err)
		}
	})

	if !strings.Contains(output, "Estimated time to completion") {
		t.Fatalf("expected estimate message, got:\n%s", output)
	}
}

func TestTimelineCmd_PrintsEvents(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	audit := application.NewAuditService(repo)
	_ = audit.Log("spec.update", "tester", nil)
	_ = audit.Log("plan.generate", "tester", map[string]interface{}{"count": 1})

	output := captureStdout(t, func() {
		if err := timelineCmd.RunE(timelineCmd, []string{}); err != nil {
			t.Fatalf("timeline failed: %v", err)
		}
	})

	if !strings.Contains(output, "Project Timeline") {
		t.Fatalf("expected timeline header, got:\n%s", output)
	}
	if !strings.Contains(output, "plan.generate") {
		t.Fatalf("expected event output, got:\n%s", output)
	}
}

func TestUsageCmd_PrintsStats(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	// Use UsageService for token tracking (SRP: audit logs events, usage tracks tokens)
	usageSvc := application.NewUsageService(repo)
	if err := usageSvc.RecordTokenUsage("gpt-4o", 5, 3); err != nil {
		t.Fatalf("record token usage: %v", err)
	}

	output := captureStdout(t, func() {
		if err := usageCmd.RunE(usageCmd, []string{}); err != nil {
			t.Fatalf("usage failed: %v", err)
		}
	})

	if !strings.Contains(output, "Project Usage Metrics") {
		t.Fatalf("expected usage header, got:\n%s", output)
	}
	if !strings.Contains(output, "gpt-4o:input") {
		t.Fatalf("expected provider stats, got:\n%s", output)
	}
}

func TestUsageCmd_NoStats(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	if err := repo.SaveSpec(&spec.ProductSpec{ID: "spec-1", Title: "Project"}); err != nil {
		t.Fatalf("save spec: %v", err)
	}
	if err := repo.UpdateUsage(domain.UsageStats{}); err != nil {
		t.Fatalf("update usage: %v", err)
	}

	output := captureStdout(t, func() {
		if err := usageCmd.RunE(usageCmd, []string{}); err != nil {
			t.Fatalf("usage failed: %v", err)
		}
	})
	if !strings.Contains(output, "Project Usage Metrics") {
		t.Fatalf("expected usage output, got:\n%s", output)
	}
}

func TestDiscoverCmd_FindsProjects(t *testing.T) {
	root, cleanup := withTempDir(t)
	defer cleanup()

	project := root + "/project-a"
	if err := os.MkdirAll(project+"/.roady", 0700); err != nil {
		t.Fatalf("create project: %v", err)
	}

	output := captureStdout(t, func() {
		if err := discoverCmd.RunE(discoverCmd, []string{root}); err != nil {
			t.Fatalf("discover failed: %v", err)
		}
	})

	if !strings.Contains(output, "Found 1 Roady projects") {
		t.Fatalf("expected discovery output, got:\n%s", output)
	}
}

func TestOrgStatusCmd_PrintsTable(t *testing.T) {
	root, cleanup := withTempDir(t)
	defer cleanup()

	project := root + "/project-a"
	repo := storage.NewFilesystemRepository(project)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	if err := repo.SaveSpec(&spec.ProductSpec{
		ID:      "spec-1",
		Title:   "Alpha",
		Version: "0.1.0",
		Features: []spec.Feature{
			{ID: "f1", Title: "Feature"},
		},
	}); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	plan := &planning.Plan{
		ID:             "p1",
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState("p1")
	state.TaskStates["t1"] = planning.TaskResult{Status: planning.StatusVerified}
	state.TaskStates["t2"] = planning.TaskResult{Status: planning.StatusInProgress}
	state.UpdatedAt = time.Now()
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	output := captureStdout(t, func() {
		if err := orgStatusCmd.RunE(orgStatusCmd, []string{root}); err != nil {
			t.Fatalf("org status failed: %v", err)
		}
	})

	if !strings.Contains(output, "Organizational Status (1 projects)") {
		t.Fatalf("expected org status header, got:\n%s", output)
	}
	if !strings.Contains(output, "Alpha") {
		t.Fatalf("expected project name, got:\n%s", output)
	}
	if !strings.Contains(output, "50.0%") {
		t.Fatalf("expected progress output, got:\n%s", output)
	}
}
