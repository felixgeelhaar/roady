package cli

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestIsTaskUnlockedByDeps(t *testing.T) {
	task := planning.Task{ID: "t3", DependsOn: []string{"t1", "t2"}}

	state := planning.NewExecutionState("plan-1")
	state.TaskStates["t1"] = planning.TaskResult{Status: planning.StatusDone}
	state.TaskStates["t2"] = planning.TaskResult{Status: planning.StatusPending}

	if isTaskUnlockedByDeps(task, state) {
		t.Fatal("expected task to be locked when dependency is incomplete")
	}

	state.TaskStates["t2"] = planning.TaskResult{Status: planning.StatusVerified}
	if !isTaskUnlockedByDeps(task, state) {
		t.Fatal("expected task to be unlocked when dependencies are complete")
	}
}
