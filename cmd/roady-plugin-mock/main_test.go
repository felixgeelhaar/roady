package main

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
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

	result, err := m.Sync(plan, state)
	if err != nil {
		t.Fatal(err)
	}

	if result.StatusUpdates["t1"] != planning.StatusDone {
		t.Errorf("Expected t1 done, got %s", result.StatusUpdates["t1"])
	}
	if result.StatusUpdates["t2"] != planning.StatusInProgress {
		t.Errorf("Expected t2 in_progress, got %s", result.StatusUpdates["t2"])
	}
}
