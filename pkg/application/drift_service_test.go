package application_test

import (
	"context"
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
	defer func() { _ = os.RemoveAll(tempDir) }()

	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	service := application.NewDriftService(repo, audit, storage.NewCodebaseInspector(), policy)

	// 1. Spec Drift (Missing Task)
	s := &spec.ProductSpec{Features: []spec.Feature{{
		ID: "f1", Title: "F1",
		Requirements: []spec.Requirement{{ID: "r1", Title: "R1"}},
	}}}
	if err := repo.SaveSpec(s); err != nil {
		t.Fatal(err)
	}

	plan := &planning.Plan{Tasks: []planning.Task{}}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatal(err)
	}

	report, err := service.DetectDrift(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 1 || report.Issues[0].Type != drift.DriftTypePlan {
		t.Errorf("Expected 1 plan drift, got %d", len(report.Issues))
	}

	// 2. Code Drift (Empty File)
	emptyFile := filepath.Join(tempDir, "empty.go")
	if err := os.WriteFile(emptyFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Add t1 as task-r1 to match requirement r1 and avoid orphan
	plan.Tasks = append(plan.Tasks, planning.Task{
		ID: "task-r1", FeatureID: "f1",
	})
	if err := repo.SavePlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveState(&planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-r1": {Status: planning.StatusDone, Path: emptyFile},
		},
	}); err != nil {
		t.Fatal(err)
	}

	report, err = service.DetectDrift(context.Background())
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
	if err := os.WriteFile(emptyFile, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	report, _ = service.DetectDrift(context.Background())
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
	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	service := application.NewDriftService(repo, audit, &MockInspector{}, policy)

	report, _ := service.DetectDrift(context.Background())
	if len(report.Issues) == 0 {
		t.Error("Expected drift")
	}
}

func TestDriftService_Errors(t *testing.T) {
	repo := &MockRepo{LoadError: errors.New("fail")}
	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	service := application.NewDriftService(repo, audit, &MockInspector{}, policy)

	_, err := service.DetectDrift(context.Background())
	if err == nil {
		t.Error("expected error on load fail")
	}
}

func TestDriftService_AcceptDrift(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-drift-accept-*")
	defer func() { _ = os.RemoveAll(tempDir) }()

	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	spec := &spec.ProductSpec{ID: "accept-spec", Title: "Accept Spec"}
	if err := repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	service := application.NewDriftService(repo, audit, storage.NewCodebaseInspector(), policy)

	if err := service.AcceptDrift(); err != nil {
		t.Fatalf("accept drift failed: %v", err)
	}

	lock, err := repo.LoadSpecLock()
	if err != nil {
		t.Fatalf("load spec lock: %v", err)
	}
	if lock == nil || lock.ID != spec.ID {
		t.Fatalf("expected locked spec, got %+v", lock)
	}

	events, err := repo.LoadEvents()
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected governance event")
	}
	last := events[len(events)-1]
	if last.Action != "drift.accepted" {
		t.Fatalf("unexpected event action: %s", last.Action)
	}
	if id, _ := last.Metadata["spec_id"]; id != spec.ID {
		t.Fatalf("unexpected metadata: %+v", last.Metadata)
	}
}
