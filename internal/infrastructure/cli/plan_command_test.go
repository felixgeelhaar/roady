package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestPlanGenerateCommandSuccess(t *testing.T) {
	tempDir := t.TempDir()
	workspace := storage.NewFilesystemRepository(tempDir)
	if err := workspace.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	spec := &spec.ProductSpec{
		ID:    "project-b",
		Title: "Project B",
		Features: []spec.Feature{
			{
				ID:          "feature-core",
				Title:       "Core Feature",
				Description: "Do core work",
				Requirements: []spec.Requirement{
					{ID: "req-1", Title: "Requirement 1", Description: "Description", Priority: "high", Estimate: "4h"},
				},
			},
		},
	}
	if err := workspace.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	output := captureStdout(t, func() {
		if err := planGenerateCmd.RunE(planGenerateCmd, []string{}); err != nil {
			t.Fatalf("generate plan failed: %v", err)
		}
	})

	if !strings.Contains(output, "Successfully generated plan") {
		t.Fatalf("expected success output, got %q", output)
	}
}

func TestPlanGenerateCommandMissingSpec(t *testing.T) {
	tempDir := t.TempDir()
	workspace := storage.NewFilesystemRepository(tempDir)
	if err := workspace.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	err := planGenerateCmd.RunE(planGenerateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when spec is missing")
	}
	if !strings.Contains(err.Error(), "load spec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanCommandsAuditTrail(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	spec := &spec.ProductSpec{
		ID:    "audit-spec",
		Title: "Audit Spec",
		Features: []spec.Feature{
			{ID: "f1", Title: "Feature One"},
		},
	}
	if err := repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	plan := &planning.Plan{
		ID:             "audit-plan",
		SpecID:         spec.ID,
		ApprovalStatus: planning.ApprovalPending,
		Tasks: []planning.Task{
			{ID: "task-f1", FeatureID: "f1", Title: "Feature Task"},
			{ID: "task-orphan", FeatureID: "missing", Title: "Orphan Task"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	if err := repo.SaveState(planning.NewExecutionState(plan.ID)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 3, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	commands := []struct {
		name string
		fn   func() error
	}{
		{"approve", func() error { return planApproveCmd.RunE(planApproveCmd, []string{}) }},
		{"reject", func() error { return planRejectCmd.RunE(planRejectCmd, []string{}) }},
		{"prune", func() error { return planPruneCmd.RunE(planPruneCmd, []string{}) }},
	}

	for _, cmd := range commands {
		output := captureStdout(t, func() {
			if err := cmd.fn(); err != nil {
				t.Fatalf("plan %s failed: %v", cmd.name, err)
			}
		})
		if output == "" {
			t.Fatalf("expected %s command to emit output", cmd.name)
		}
	}

	events, err := repo.LoadEvents()
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	found := map[string]bool{}
	for _, ev := range events {
		found[ev.Action] = true
	}

	for _, want := range []string{"plan.approved", "plan.reject", "plan.prune"} {
		if !found[want] {
			t.Fatalf("expected governance event %s, got events: %+v", want, events)
		}
	}
}
