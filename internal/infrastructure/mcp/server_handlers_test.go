package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestServer_HandleTransitionalTools(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	spec := &spec.ProductSpec{
		ID:    "mcp",
		Title: "MCP Project",
		Features: []spec.Feature{
			{ID: "feature-1", Title: "Feature 1", Requirements: []spec.Requirement{{ID: "req-1", Title: "Req 1"}}},
		},
	}
	if err := repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}
	plan := &planning.Plan{
		ID:             "plan-1",
		SpecID:         spec.ID,
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "task-req-1", Title: "Task 1", FeatureID: "feature-1"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	if err := repo.SavePolicy(&domain.PolicyConfig{AllowAI: true, MaxWIP: 2}); err != nil {
		t.Fatalf("save policy: %v", err)
	}
	if err := repo.UpdateUsage(domain.UsageStats{}); err != nil {
		t.Fatalf("update usage: %v", err)
	}

	server, err := NewServer(tempDir)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	ctx := context.Background()

	if _, err := server.handleGetUsage(ctx, struct{}{}); err != nil {
		t.Fatalf("handleGetUsage failed: %v", err)
	}

	if _, err := server.handleGetPlan(ctx, struct{}{}); err != nil {
		t.Fatalf("handleGetPlan failed: %v", err)
	}

	if _, err := server.handleGetState(ctx, struct{}{}); err != nil {
		t.Fatalf("handleGetState failed: %v", err)
	}

	if _, err := server.handleTransitionTask(ctx, TransitionTaskArgs{
		TaskID:   "task-req-1",
		Event:    "start",
		Evidence: "test",
	}); err != nil {
		t.Fatalf("handleTransitionTask failed: %v", err)
	}

	if _, err := server.handleCheckPolicy(ctx, struct{}{}); err != nil {
		t.Fatalf("handleCheckPolicy failed: %v", err)
	}

	if _, err := server.handleDetectDrift(ctx, struct{}{}); err != nil {
		t.Fatalf("handleDetectDrift failed: %v", err)
	}

	if _, err := server.handleAcceptDrift(ctx, struct{}{}); err != nil {
		t.Fatalf("handleAcceptDrift failed: %v", err)
	}
}

func TestServerHandleStatusCounts(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	spec := &spec.ProductSpec{
		ID:    "status-spec",
		Title: "Status Spec",
		Features: []spec.Feature{
			{ID: "feat-a", Title: "Feature A"},
		},
	}
	if err := repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	plan := &planning.Plan{
		ID:             "plan-status",
		SpecID:         spec.ID,
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "task-done", FeatureID: "feat-a", Title: "Done Task"},
			{ID: "task-progress", FeatureID: "feat-a", Title: "In Progress Task"},
			{ID: "task-pending", FeatureID: "feat-a", Title: "Pending Task"},
			{ID: "task-blocked", FeatureID: "feat-a", Title: "Blocked Task"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState(plan.ID)
	state.TaskStates["task-done"] = planning.TaskResult{Status: planning.StatusDone}
	state.TaskStates["task-progress"] = planning.TaskResult{Status: planning.StatusInProgress}
	state.TaskStates["task-pending"] = planning.TaskResult{Status: planning.StatusPending}
	state.TaskStates["task-blocked"] = planning.TaskResult{Status: planning.StatusBlocked}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	status, err := server.handleStatus(context.Background(), struct{}{})
	if err != nil {
		t.Fatalf("handleStatus failed: %v", err)
	}

	for _, want := range []string{"- Done: 1", "- In Progress: 1", "- Pending: 1", "- Blocked: 1"} {
		if !strings.Contains(status, want) {
			t.Fatalf("expected %q in status summary, got:\n%s", want, status)
		}
	}
}

func TestServerHandleCheckPolicyViolations(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	spec := &spec.ProductSpec{
		ID:    "policy-spec",
		Title: "Policy Project",
		Features: []spec.Feature{
			{
				ID:    "feature-pol",
				Title: "Policy Feature",
				Requirements: []spec.Requirement{
					{ID: "req-pol", Title: "Req", Description: "Desc"},
				},
			},
		},
	}
	if err := repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	plan := &planning.Plan{
		ID:             "plan-policy",
		SpecID:         spec.ID,
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "task-a", FeatureID: "feature-pol", Title: "Task A"},
			{ID: "task-b", FeatureID: "feature-pol", Title: "Task B"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState(plan.ID)
	state.TaskStates["task-a"] = planning.TaskResult{Status: planning.StatusInProgress}
	state.TaskStates["task-b"] = planning.TaskResult{Status: planning.StatusInProgress}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 1, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	result, err := server.handleCheckPolicy(context.Background(), struct{}{})
	if err != nil {
		t.Fatalf("handleCheckPolicy failed: %v", err)
	}

	violations, ok := result.([]policy.Violation)
	if !ok {
		t.Fatalf("expected []policy.Violation, got %T", result)
	}
	if len(violations) == 0 {
		t.Fatalf("expected violations, got none")
	}
}
