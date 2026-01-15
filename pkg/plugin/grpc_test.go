package plugin

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
)

func TestPlanToProto(t *testing.T) {
	plan := &planning.Plan{
		ID:     "plan-123",
		SpecID: "spec-456",
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Test Task", Description: "A test task"},
		},
		ApprovalStatus: planning.ApprovalApproved,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	proto := planToProto(plan)

	if proto.Id != plan.ID {
		t.Errorf("Expected ID %s, got %s", plan.ID, proto.Id)
	}
	if proto.SpecId != plan.SpecID {
		t.Errorf("Expected SpecID %s, got %s", plan.SpecID, proto.SpecId)
	}
	if len(proto.Tasks) != len(plan.Tasks) {
		t.Errorf("Expected %d tasks, got %d", len(plan.Tasks), len(proto.Tasks))
	}
	if proto.ApprovalStatus != string(plan.ApprovalStatus) {
		t.Errorf("Expected approval status %s, got %s", plan.ApprovalStatus, proto.ApprovalStatus)
	}
}

func TestPlanFromProto(t *testing.T) {
	proto := planToProto(&planning.Plan{
		ID:             "plan-789",
		SpecID:         "spec-012",
		Tasks:          []planning.Task{{ID: "task-2", Title: "Task 2"}},
		ApprovalStatus: planning.ApprovalPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	plan := planFromProto(proto)

	if plan.ID != proto.Id {
		t.Errorf("Expected ID %s, got %s", proto.Id, plan.ID)
	}
	if plan.SpecID != proto.SpecId {
		t.Errorf("Expected SpecID %s, got %s", proto.SpecId, plan.SpecID)
	}
	if len(plan.Tasks) != len(proto.Tasks) {
		t.Errorf("Expected %d tasks, got %d", len(proto.Tasks), len(plan.Tasks))
	}
}

func TestTaskToProto(t *testing.T) {
	task := planning.Task{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "A test task description",
		Priority:    planning.PriorityHigh,
		DependsOn:   []string{"task-0"},
	}

	proto := taskToProto(task)

	if proto.Id != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, proto.Id)
	}
	if proto.Title != task.Title {
		t.Errorf("Expected Title %s, got %s", task.Title, proto.Title)
	}
	if proto.Description != task.Description {
		t.Errorf("Expected Description %s, got %s", task.Description, proto.Description)
	}
	if len(proto.Dependencies) != len(task.DependsOn) {
		t.Errorf("Expected %d dependencies, got %d", len(task.DependsOn), len(proto.Dependencies))
	}
}

func TestTaskFromProto(t *testing.T) {
	proto := taskToProto(planning.Task{
		ID:          "task-abc",
		Title:       "Proto Task",
		Description: "Description from proto",
		Priority:    planning.PriorityMedium,
		DependsOn:   []string{"dep-1", "dep-2"},
	})

	task := taskFromProto(proto)

	if task.ID != proto.Id {
		t.Errorf("Expected ID %s, got %s", proto.Id, task.ID)
	}
	if task.Title != proto.Title {
		t.Errorf("Expected Title %s, got %s", proto.Title, task.Title)
	}
	if len(task.DependsOn) != len(proto.Dependencies) {
		t.Errorf("Expected %d dependencies, got %d", len(proto.Dependencies), len(task.DependsOn))
	}
}

func TestStateToProto(t *testing.T) {
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {
				Status: planning.StatusInProgress,
				Owner:  "alice",
				ExternalRefs: map[string]planning.ExternalRef{
					"github": {ID: "123", Identifier: "#123", URL: "https://github.com/..."},
				},
			},
		},
	}

	proto := stateToProto(state)

	if len(proto.TaskStates) != len(state.TaskStates) {
		t.Errorf("Expected %d task states, got %d", len(state.TaskStates), len(proto.TaskStates))
	}

	taskState := proto.TaskStates["task-1"]
	if taskState.Status != string(planning.StatusInProgress) {
		t.Errorf("Expected status %s, got %s", planning.StatusInProgress, taskState.Status)
	}
	if taskState.Owner != "alice" {
		t.Errorf("Expected owner alice, got %s", taskState.Owner)
	}
}

func TestStateFromProto(t *testing.T) {
	proto := stateToProto(&planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-2": {
				Status:       planning.StatusDone,
				Owner:        "bob",
				ExternalRefs: map[string]planning.ExternalRef{},
			},
		},
	})

	state := stateFromProto(proto)

	if len(state.TaskStates) != len(proto.TaskStates) {
		t.Errorf("Expected %d task states, got %d", len(proto.TaskStates), len(state.TaskStates))
	}

	taskState := state.TaskStates["task-2"]
	if taskState.Status != planning.StatusDone {
		t.Errorf("Expected status done, got %s", taskState.Status)
	}
	if taskState.Owner != "bob" {
		t.Errorf("Expected owner bob, got %s", taskState.Owner)
	}
}

func TestSyncResultToProto(t *testing.T) {
	result := &domainPlugin.SyncResult{
		StatusUpdates: map[string]planning.TaskStatus{
			"task-1": planning.StatusDone,
			"task-2": planning.StatusInProgress,
		},
		LinkUpdates: map[string]planning.ExternalRef{
			"task-3": {ID: "456", Identifier: "#456"},
		},
		Errors: []string{"error1", "error2"},
	}

	proto := syncResultToProto(result)

	if len(proto.StatusUpdates) != len(result.StatusUpdates) {
		t.Errorf("Expected %d status updates, got %d", len(result.StatusUpdates), len(proto.StatusUpdates))
	}
	if len(proto.LinkUpdates) != len(result.LinkUpdates) {
		t.Errorf("Expected %d link updates, got %d", len(result.LinkUpdates), len(proto.LinkUpdates))
	}
	if len(proto.Errors) != len(result.Errors) {
		t.Errorf("Expected %d errors, got %d", len(result.Errors), len(proto.Errors))
	}
}

func TestSyncResultFromProto(t *testing.T) {
	proto := syncResultToProto(&domainPlugin.SyncResult{
		StatusUpdates: map[string]planning.TaskStatus{
			"task-x": planning.StatusPending,
		},
		LinkUpdates: map[string]planning.ExternalRef{
			"task-y": {ID: "789"},
		},
		Errors: []string{},
	})

	result := syncResultFromProto(proto)

	if len(result.StatusUpdates) != len(proto.StatusUpdates) {
		t.Errorf("Expected %d status updates, got %d", len(proto.StatusUpdates), len(result.StatusUpdates))
	}
	if result.StatusUpdates["task-x"] != planning.StatusPending {
		t.Errorf("Expected pending status, got %s", result.StatusUpdates["task-x"])
	}
}

func TestExternalRefRoundTrip(t *testing.T) {
	ref := planning.ExternalRef{
		ID:           "ext-123",
		Identifier:   "EXT-123",
		URL:          "https://example.com/ext/123",
		LastSyncedAt: time.Now().Truncate(time.Second),
	}

	proto := externalRefToProto(ref)
	result := externalRefFromProto(proto)

	if result.ID != ref.ID {
		t.Errorf("Expected ID %s, got %s", ref.ID, result.ID)
	}
	if result.Identifier != ref.Identifier {
		t.Errorf("Expected Identifier %s, got %s", ref.Identifier, result.Identifier)
	}
	if result.URL != ref.URL {
		t.Errorf("Expected URL %s, got %s", ref.URL, result.URL)
	}
	// Compare time with some tolerance for serialization
	if result.LastSyncedAt.Sub(ref.LastSyncedAt) > time.Second {
		t.Errorf("LastSyncedAt mismatch: %v vs %v", ref.LastSyncedAt, result.LastSyncedAt)
	}
}

func TestNilHandling(t *testing.T) {
	// Test nil plan
	if planToProto(nil) != nil {
		t.Error("Expected nil for nil plan")
	}
	if planFromProto(nil) != nil {
		t.Error("Expected nil for nil proto plan")
	}

	// Test nil state
	if stateToProto(nil) != nil {
		t.Error("Expected nil for nil state")
	}
	if stateFromProto(nil) != nil {
		t.Error("Expected nil for nil proto state")
	}
}
