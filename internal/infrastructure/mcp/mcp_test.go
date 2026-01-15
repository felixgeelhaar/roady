package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestServer_Handlers(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-mcp-handlers-*")
	defer os.RemoveAll(tempDir)

	s, err := NewServer(tempDir)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	// 1. HandleInit
	resp, err := s.handleInit(context.Background(), InitArgs{Name: "test"})
	if err != nil {
		t.Errorf("handleInit failed: %v", err)
	}
	if resp != "Project test initialized successfully" {
		t.Errorf("unexpected response: %v", resp)
	}

	// 1.1 HandleInit Error (empty name)
	_, err = s.handleInit(context.Background(), InitArgs{Name: ""})
	if err == nil {
		t.Error("expected error for empty name")
	}

	// 2. HandleGeneratePlan
	_, err = s.handleGeneratePlan(context.Background(), struct{}{})
	if err != nil {
		t.Errorf("handleGeneratePlan failed: %v", err)
	}

	// 3. HandleGetPlan
	_, err = s.handleGetPlan(context.Background(), struct{}{})
	if err != nil {
		t.Errorf("handleGetPlan failed: %v", err)
	}

	// 3.1 HandleGetSpec
	repo := storage.NewFilesystemRepository(tempDir)
	repo.SaveSpec(&spec.ProductSpec{ID: "s1", Title: "S1"})
	_, err = s.handleGetSpec(context.Background(), struct{}{})
	if err != nil {
		t.Errorf("handleGetSpec failed: %v", err)
	}

	// 4. HandleUpdatePlan
	_, err = s.handleUpdatePlan(context.Background(), UpdatePlanArgs{
		Tasks: []planning.Task{
			{ID: "t1", Title: "T1", FeatureID: "f1"},
			{ID: "t2", Title: "T2", FeatureID: "f1"},
		},
	})
	if err != nil {
		t.Errorf("handleUpdatePlan failed: %v", err)
	}

	// 4.1 HandleUpdatePlan empty
	_, err = s.handleUpdatePlan(context.Background(), UpdatePlanArgs{Tasks: []planning.Task{}})
	if err != nil {
		t.Errorf("handleUpdatePlan empty failed: %v", err)
	}

	// 5. HandleDetectDrift
	repo = storage.NewFilesystemRepository(tempDir)
	repo.SaveSpec(&spec.ProductSpec{ID: "s1", Title: "S1", Features: []spec.Feature{{ID: "f1"}}})
	// Provide plan to avoid drift or show it
	repo.SavePlan(&planning.Plan{Tasks: []planning.Task{{ID: "task-f1", FeatureID: "f1"}}})

	_, err = s.handleDetectDrift(context.Background(), struct{}{})
	if err != nil {
		t.Errorf("handleDetectDrift failed: %v", err)
	}

	// 6. HandleStatus
	repo.SavePlan(&planning.Plan{Tasks: []planning.Task{
		{ID: "t1"},
		{ID: "t2"},
	}})
	repo.SaveState(&planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"t1": {Status: planning.StatusInProgress},
			"t2": {Status: planning.StatusDone},
		},
	})
	_, err = s.handleStatus(context.Background(), struct{}{})
	if err != nil {
		t.Errorf("handleStatus failed: %v", err)
	}

	// 7. HandleCheckPolicy
	// Success path
	repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 10, AllowAI: true})
	_, err = s.handleCheckPolicy(context.Background(), struct{}{})
	if err != nil {
		t.Errorf("handleCheckPolicy failed: %v", err)
	}

	// Force violation
	repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 1, AllowAI: true})
	_, err = s.handleCheckPolicy(context.Background(), struct{}{})
	if err != nil {
		t.Errorf("handleCheckPolicy failed: %v", err)
	}

	// 8. Error cases (restricted dir)
	os.Chmod(tempDir+"/.roady", 0000)
	defer os.Chmod(tempDir+"/.roady", 0700)

	_, err = s.handleGetPlan(context.Background(), struct{}{})
	if err == nil {
		t.Error("expected error for handleGetPlan on restricted dir")
	}

	// 8.1 HandleGeneratePlan missing spec
	tempEmpty2, _ := os.MkdirTemp("", "roady-mcp-empty-*")
	defer os.RemoveAll(tempEmpty2)
	s2, err := NewServer(tempEmpty2)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	_, err = s2.handleGeneratePlan(context.Background(), struct{}{})
	if err == nil {
		t.Error("expected error for handleGeneratePlan without spec")
	}

	// 8.1.1 HandleGeneratePlan error (restricted dir)
	_, err = s.handleGeneratePlan(context.Background(), struct{}{})
	if err == nil {
		t.Error("expected error for handleGeneratePlan on restricted dir")
	}

	// 8.2.1 HandleGetSpec error (restricted dir)
	_, err = s.handleGetSpec(context.Background(), struct{}{})
	if err == nil {
		t.Error("expected error for handleGetSpec on restricted dir")
	}

	// 8.3 HandleUpdatePlan error (cycle)

	_, err = s.handleUpdatePlan(context.Background(), UpdatePlanArgs{

		Tasks: []planning.Task{{ID: "t1", DependsOn: []string{"t1"}}},
	})

	if err == nil {

		t.Error("expected error for handleUpdatePlan with cycle")

	}

	// 8.3.1 HandleUpdatePlan error (restricted dir)

	_, err = s.handleUpdatePlan(context.Background(), UpdatePlanArgs{

		Tasks: []planning.Task{{ID: "t1"}},
	})

	if err == nil {

		t.Error("expected error for handleUpdatePlan on restricted dir")

	}

	// 8.4 HandleStatus error
	_, err = s.handleStatus(context.Background(), struct{}{})
	if err == nil {
		t.Error("expected error for handleStatus on restricted dir")
	}

	// 8.5 HandleDetectDrift error
	_, err = s.handleDetectDrift(context.Background(), struct{}{})
	if err == nil {
		t.Error("expected error for handleDetectDrift on restricted dir")
	}

	// 8.6 HandleCheckPolicy error
	_, err = s.handleCheckPolicy(context.Background(), struct{}{})
	if err == nil {
		t.Error("expected error for handleCheckPolicy on restricted dir")
	}
}
