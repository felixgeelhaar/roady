package application_test

import (
	"errors"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestTaskService_Transition_Mock(t *testing.T) {
	repo := &MockRepo{
		Plan: &planning.Plan{
			Tasks:          []planning.Task{{ID: "t1"}},
			ApprovalStatus: planning.ApprovalApproved,
		},
		State: &planning.ExecutionState{
			TaskStates: map[string]planning.TaskResult{
				"t1": {Status: planning.StatusPending},
			},
		},
	}
	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	service := application.NewTaskService(repo, audit, policy)

	// 1. Valid
	err := service.TransitionTask("t1", "start", "test-user", "some evidence")
	if err != nil {
		t.Fatal(err)
	}
	if repo.State.TaskStates["t1"].Status != planning.StatusInProgress {
		t.Error("Expected InProgress")
	}

	// 2. Task not found
	err = service.TransitionTask("missing", "start", "test-user", "")
	if err == nil {
		t.Error("Expected error for missing task")
	}

	// 3. Save error
	repo.SaveError = errors.New("save fail")
	err = service.TransitionTask("t1", "start", "test-user", "")
	if err == nil {
		t.Error("Expected error on save fail")
	}
}

func TestTaskService_LinkTask(t *testing.T) {
	repo := &MockRepo{
		State: &planning.ExecutionState{
			TaskStates: map[string]planning.TaskResult{
				"t1": {Status: planning.StatusPending},
			},
		},
	}
	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	service := application.NewTaskService(repo, audit, policy)

	ref := planning.ExternalRef{
		ID:         "123",
		Identifier: "EXT-1",
		URL:        "https://example.com/EXT-1",
	}
	if err := service.LinkTask("t1", "jira", ref); err != nil {
		t.Fatalf("LinkTask failed: %v", err)
	}
	if repo.State.TaskStates["t1"].ExternalRefs["jira"].Identifier != "EXT-1" {
		t.Fatalf("expected external ref to be stored")
	}
}
