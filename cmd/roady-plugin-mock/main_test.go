package main

import (
	"testing"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

func TestMockSyncer_Sync(t *testing.T) {
	m := &MockSyncer{}
	plan := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "t1"},
			{ID: "t2"},
		},
	}
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"t1": {Status: planning.StatusInProgress},
			"t2": {Status: planning.StatusPending},
		},
	}

	updates, err := m.Sync(plan, state)
	if err != nil {
		t.Fatal(err)
	}
	
	if updates["t1"] != planning.StatusDone {
		t.Errorf("Expected t1 done, got %s", updates["t1"])
	}
	if updates["t2"] != planning.StatusInProgress {
		t.Errorf("Expected t2 in_progress, got %s", updates["t2"])
	}
}
