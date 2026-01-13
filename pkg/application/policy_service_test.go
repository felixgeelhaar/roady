package application_test

import (
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
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
