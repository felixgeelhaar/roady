package application_test

import (
	"errors"
	"testing"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
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
	service := application.NewTaskService(repo)

	// 1. Valid
	err := service.TransitionTask("t1", "start")
	if err != nil {
		t.Fatal(err)
	}
	if repo.State.TaskStates["t1"].Status != planning.StatusInProgress {
		t.Error("Expected InProgress")
	}

	// 2. Task not found
	err = service.TransitionTask("missing", "start")
		if err == nil {
			t.Error("Expected error for missing task")
		}
	
		// 3. Save error
		repo.SaveError = errors.New("save fail")
		err = service.TransitionTask("t1", "start")
		if err == nil {
			t.Error("Expected error on save fail")
		}
	}
	