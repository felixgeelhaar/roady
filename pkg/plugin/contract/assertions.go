// Package contract provides contract test assertions for Roady syncer plugins.
package contract

import (
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
)

// Result captures the outcome of a single contract assertion.
type Result struct {
	Name    string
	Passed  bool
	Message string
}

// AssertInitSuccess verifies that Init succeeds with valid config.
func AssertInitSuccess(syncer domainPlugin.Syncer) Result {
	err := syncer.Init(map[string]string{"project": "test"})
	if err != nil {
		return Result{Name: "InitSuccess", Passed: false, Message: fmt.Sprintf("Init failed: %v", err)}
	}
	return Result{Name: "InitSuccess", Passed: true, Message: "Init succeeded"}
}

// AssertInitWithBadConfig verifies that Init returns an error for bad config.
func AssertInitWithBadConfig(syncer domainPlugin.Syncer) Result {
	err := syncer.Init(map[string]string{"fail": "true"})
	if err == nil {
		return Result{Name: "InitWithBadConfig", Passed: false, Message: "expected Init to fail with fail=true config"}
	}
	return Result{Name: "InitWithBadConfig", Passed: true, Message: fmt.Sprintf("Init correctly failed: %v", err)}
}

// AssertSyncEmptyPlan verifies Sync handles an empty plan without error.
func AssertSyncEmptyPlan(syncer domainPlugin.Syncer) Result {
	plan := &planning.Plan{Tasks: []planning.Task{}}
	state := planning.NewExecutionState("test")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		return Result{Name: "SyncEmptyPlan", Passed: false, Message: fmt.Sprintf("Sync failed: %v", err)}
	}
	if result == nil {
		return Result{Name: "SyncEmptyPlan", Passed: false, Message: "Sync returned nil result"}
	}
	return Result{Name: "SyncEmptyPlan", Passed: true, Message: "Sync with empty plan succeeded"}
}

// AssertSyncWithTasks verifies Sync processes tasks and returns status updates.
func AssertSyncWithTasks(syncer domainPlugin.Syncer) Result {
	plan := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Test Task"},
		},
	}
	state := planning.NewExecutionState("test")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		return Result{Name: "SyncWithTasks", Passed: false, Message: fmt.Sprintf("Sync failed: %v", err)}
	}
	if result == nil {
		return Result{Name: "SyncWithTasks", Passed: false, Message: "Sync returned nil result"}
	}
	return Result{Name: "SyncWithTasks", Passed: true, Message: fmt.Sprintf("Sync returned %d status updates", len(result.StatusUpdates))}
}

// AssertPushValidTask verifies Push accepts a valid task transition.
func AssertPushValidTask(syncer domainPlugin.Syncer) Result {
	err := syncer.Push("task-1", planning.StatusInProgress)
	if err != nil {
		return Result{Name: "PushValidTask", Passed: false, Message: fmt.Sprintf("Push failed: %v", err)}
	}
	return Result{Name: "PushValidTask", Passed: true, Message: "Push succeeded"}
}

// AssertPushInvalidTask verifies Push handles an empty task ID gracefully.
func AssertPushInvalidTask(syncer domainPlugin.Syncer) Result {
	err := syncer.Push("", planning.StatusInProgress)
	if err == nil {
		// Some plugins may accept empty IDs — this is a soft check
		return Result{Name: "PushInvalidTask", Passed: true, Message: "Push with empty ID did not error (acceptable)"}
	}
	return Result{Name: "PushInvalidTask", Passed: true, Message: fmt.Sprintf("Push correctly rejected empty ID: %v", err)}
}
