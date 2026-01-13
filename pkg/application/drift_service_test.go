package application_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/drift"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestDriftService_Detect(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-drift-test-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	service := application.NewDriftService(repo)

	// 1. Spec Drift (Missing Task)
	s := &spec.ProductSpec{Features: []spec.Feature{{
		ID: "f1", Title: "F1",
		Requirements: []spec.Requirement{{ID: "r1", Title: "R1"}},
	}}}
	repo.SaveSpec(s)
	
	plan := &planning.Plan{Tasks: []planning.Task{}}
	repo.SavePlan(plan)

	report, err := service.DetectDrift()
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 1 || report.Issues[0].Type != drift.DriftTypePlan {
		t.Errorf("Expected 1 plan drift, got %d", len(report.Issues))
	}

	// 2. Code Drift (Empty File)
	emptyFile := filepath.Join(tempDir, "empty.go")
	os.WriteFile(emptyFile, []byte(""), 0600)

	// Add t1 as task-r1 to match requirement r1 and avoid orphan
	plan.Tasks = append(plan.Tasks, planning.Task{
		ID: "task-r1", FeatureID: "f1",
	})
	repo.SavePlan(plan)
	repo.SaveState(&planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-r1": {Status: planning.StatusDone, Path: emptyFile},
		},
	})

	report, err = service.DetectDrift()
	if err != nil {
		t.Fatal(err)
	}
	// We might have 0 or more issues depending on if we still consider it missing r1
	// Wait, task-r1 matches task-r1 in DetectDrift
	found := false
	for _, iss := range report.Issues {
		if iss.ID == "empty-code-task-r1" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected empty code drift for task-r1, got %+v", report.Issues)
	}

	// 3. Normal code drift (file exists and not empty)
	os.WriteFile(emptyFile, []byte("content"), 0600)
	report, _ = service.DetectDrift()
	// Should have 0 issues now because task-r1 matches r1 and code is present
	if len(report.Issues) != 0 {
		t.Errorf("Expected 0 issues for valid implementation, got %+v", report.Issues)
	}
}

func TestDriftService_Detect_Mock(t *testing.T) {
	repo := &MockRepo{
		Spec:  &spec.ProductSpec{Features: []spec.Feature{{ID: "f1", Requirements: []spec.Requirement{{ID: "r1"}}}}},
		Plan:  &planning.Plan{Tasks: []planning.Task{}},
		State: &planning.ExecutionState{},
	}
	service := application.NewDriftService(repo)

	report, _ := service.DetectDrift()
	if len(report.Issues) == 0 {
		t.Error("Expected drift")
	}
}

func TestDriftService_Errors(t *testing.T) {
	repo := &MockRepo{LoadError: errors.New("fail")}
	service := application.NewDriftService(repo)

	_, err := service.DetectDrift()
	if err == nil {
		t.Error("expected error on load fail")
	}
}