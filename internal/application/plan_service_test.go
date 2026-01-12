package application_test

import (
	"errors"
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/domain/spec"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
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
		{ID: "t1", DependsOn: []string{"t1"}},
	}
	_, err = service.UpdatePlan(tasks)
	if err == nil {
		t.Error("Expected error for DAG cycle")
	}
}