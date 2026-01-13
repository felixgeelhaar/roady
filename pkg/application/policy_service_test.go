package application_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestPolicyService_CheckCompliance(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-policy-test-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()

	// Setup: Policy with MaxWIP = 1
	repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 1})

	// Setup: Plan with 2 InProgress tasks
	plan := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "t1"},
			{ID: "t2"},
		},
	}
	repo.SavePlan(plan)
	repo.SaveState(&planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"t1": {Status: planning.StatusInProgress},
			"t2": {Status: planning.StatusInProgress},
		},
	})

	service := application.NewPolicyService(repo)
	violations, err := service.CheckCompliance()
	if err != nil {
		t.Fatal(err)
	}

	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
	}
}

func TestPolicyService_ValidateTransition_WIP(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{MaxWIP: 1},
		Plan: &planning.Plan{
			Tasks: []planning.Task{
				{ID: "t1"},
				{ID: "t2"},
			},
		},
		State: &planning.ExecutionState{
			TaskStates: map[string]planning.TaskResult{
				"t2": {Status: planning.StatusInProgress},
			},
		},
	}
	service := application.NewPolicyService(repo)
	if err := service.ValidateTransition("t1", "start"); err == nil {
		t.Fatal("expected WIP limit error")
	}
}

func TestPolicyService_ValidateTransition_Dependency(t *testing.T) {
	repo := &MockRepo{
		Plan: &planning.Plan{
			Tasks: []planning.Task{
				{ID: "t1", DependsOn: []string{"t2"}},
				{ID: "t2"},
			},
		},
		State: &planning.ExecutionState{
			TaskStates: map[string]planning.TaskResult{
				"t2": {Status: planning.StatusPending},
			},
		},
	}
	service := application.NewPolicyService(repo)
	if err := service.ValidateTransition("t1", "start"); err == nil {
		t.Fatal("expected dependency error")
	}
}

func TestPolicyService_ValidateTransition_ExternalDependency(t *testing.T) {
	root, err := os.MkdirTemp("", "roady-policy-ext-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(root)

	projectA := filepath.Join(root, "project-a")
	projectB := filepath.Join(root, "project-b")

	repoA := storage.NewFilesystemRepository(projectA)
	repoB := storage.NewFilesystemRepository(projectB)
	_ = repoA.Initialize()
	_ = repoB.Initialize()

	if err := repoB.SaveSpec(&spec.ProductSpec{
		ID:      "ext",
		Title:   "External",
		Version: "0.1.0",
		Features: []spec.Feature{
			{ID: "f1", Title: "Feature"},
		},
	}); err != nil {
		t.Fatalf("save external spec: %v", err)
	}
	extState := planning.NewExecutionState("ext")
	extState.TaskStates["task-1"] = planning.TaskResult{Status: planning.StatusDone}
	if err := repoB.SaveState(extState); err != nil {
		t.Fatalf("save external state: %v", err)
	}

	if err := repoA.SavePlan(&planning.Plan{
		Tasks: []planning.Task{
			{ID: "t1", DependsOn: []string{"ext:task-1"}},
		},
	}); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	if err := repoA.SaveState(planning.NewExecutionState("a")); err != nil {
		t.Fatalf("save state: %v", err)
	}

	oldWD, _ := os.Getwd()
	defer os.Chdir(oldWD)
	_ = os.Chdir(projectA)

	service := application.NewPolicyService(repoA)
	if err := service.ValidateTransition("t1", "start"); err != nil {
		t.Fatalf("expected external dependency to pass, got %v", err)
	}
}
