package application_test

import (
	"errors"
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestPlanService_FullCoverage(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-plan-cov-*")
	defer os.RemoveAll(tempDir)
	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	audit := application.NewAuditService(repo)
	service := application.NewPlanService(repo, audit)

	// 1. Success Path
	repo.SaveSpec(&spec.ProductSpec{ID: "s1", Features: []spec.Feature{{ID: "f1", Title: "F1"}}})
	p, err := service.GeneratePlan()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(p.Tasks))
	}

	// 2. Reconciliation Path
	repo.SaveState(&planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			p.Tasks[0].ID: {Status: planning.StatusDone},
		},
	})
	repo.SavePlan(p)

	// Add f2
	repo.SaveSpec(&spec.ProductSpec{ID: "s1", Features: []spec.Feature{{ID: "f1", Title: "F1"}, {ID: "f2", Title: "F2"}}})
	p2, _ := service.GeneratePlan()
	state, _ := repo.LoadState()
	if len(p2.Tasks) != 2 || state.TaskStates[p2.Tasks[0].ID].Status != planning.StatusDone {
		t.Error("Reconciliation failed")
	}

	// 3. GetPlan
	gp, _ := service.GetPlan()
	if gp.ID != p2.ID {
		t.Error("GetPlan failed")
	}
}

func TestPlanService_FailurePaths(t *testing.T) {
	repo := &MockRepo{LoadError: errors.New("fail")}
	audit := application.NewAuditService(repo)
	service := application.NewPlanService(repo, audit)

	// 1. Load Spec Fail
	_, err := service.GeneratePlan()
	if err == nil {
		t.Error("Expected error on spec load fail")
	}

	// 2. Reconcile Plan with Cycle
	repo.LoadError = nil
	repo.Spec = &spec.ProductSpec{ID: "s1"}
	tasks := []planning.Task{
		{ID: "t1", Title: "T1", DependsOn: []string{"t1"}},
	}
	_, err = service.UpdatePlan(tasks)
	if err == nil {
		t.Error("Expected error for DAG cycle")
	}
}

func TestPlanService_ApproveRejectPlan(t *testing.T) {
	repo := &MockRepo{
		Plan: &planning.Plan{
			ID:             "p1",
			ApprovalStatus: planning.ApprovalPending,
		},
	}
	audit := application.NewAuditService(repo)
	service := application.NewPlanService(repo, audit)

	if err := service.ApprovePlan(); err != nil {
		t.Fatalf("ApprovePlan failed: %v", err)
	}
	if repo.Plan.ApprovalStatus != planning.ApprovalApproved {
		t.Fatalf("expected approved status, got %s", repo.Plan.ApprovalStatus)
	}

	if err := service.RejectPlan(); err != nil {
		t.Fatalf("RejectPlan failed: %v", err)
	}
	if repo.Plan.ApprovalStatus != planning.ApprovalRejected {
		t.Fatalf("expected rejected status, got %s", repo.Plan.ApprovalStatus)
	}
}

func TestPlanService_PrunePlan(t *testing.T) {
	repo := &MockRepo{
		Spec: &spec.ProductSpec{
			ID: "s1",
			Features: []spec.Feature{
				{ID: "f1", Title: "Feature 1", Requirements: []spec.Requirement{{ID: "r1", Title: "Req 1"}}},
			},
		},
		Plan: &planning.Plan{
			ID: "p1",
			Tasks: []planning.Task{
				{ID: "task-r1", FeatureID: "f1", Title: "Valid Task"},
				{ID: "task-r2", FeatureID: "f2", Title: "Invalid Task"},
			},
		},
	}
	audit := application.NewAuditService(repo)
	service := application.NewPlanService(repo, audit)

	if err := service.PrunePlan(); err != nil {
		t.Fatalf("PrunePlan failed: %v", err)
	}
	if len(repo.Plan.Tasks) != 1 {
		t.Fatalf("expected 1 task after prune, got %d", len(repo.Plan.Tasks))
	}
	if repo.Plan.Tasks[0].ID != "task-r1" {
		t.Fatalf("expected task-r1 to remain, got %s", repo.Plan.Tasks[0].ID)
	}
}

func TestPlanService_GetStateUsage(t *testing.T) {
	repo := &MockRepo{
		State: planning.NewExecutionState("p1"),
	}
	audit := application.NewAuditService(repo)
	service := application.NewPlanService(repo, audit)

	if _, err := service.GetState(); err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if _, err := service.GetUsage(); err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}
}
