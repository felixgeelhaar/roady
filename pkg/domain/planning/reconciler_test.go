package planning_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestPlanReconciler_Reconcile(t *testing.T) {
	reconciler := planning.NewPlanReconciler()

	// Fresh plan with no existing
	proposed := []planning.Task{
		{ID: "t1", Title: "Task 1"},
		{ID: "t2", Title: "Task 2"},
	}

	plan, err := reconciler.Reconcile(nil, proposed, planning.ReconcileOptions{
		SpecID: "spec-1",
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if len(plan.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(plan.Tasks))
	}
	if plan.SpecID != "spec-1" {
		t.Errorf("expected spec ID spec-1, got %s", plan.SpecID)
	}
	if plan.ApprovalStatus != planning.ApprovalPending {
		t.Errorf("expected pending approval status, got %s", plan.ApprovalStatus)
	}
}

func TestPlanReconciler_ReconcilePreservesOrphans(t *testing.T) {
	reconciler := planning.NewPlanReconciler()

	existing := &planning.Plan{
		ID:     "existing-plan",
		SpecID: "spec-1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Orphan Task"},
		},
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	// Only propose t1, t2 should be preserved as orphan
	proposed := []planning.Task{
		{ID: "t1", Title: "Task 1 Updated"},
	}

	plan, err := reconciler.Reconcile(existing, proposed, planning.ReconcileOptions{
		SpecID: "spec-1",
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("expected 2 tasks (1 proposed + 1 orphan), got %d", len(plan.Tasks))
	}

	// Should preserve existing plan ID
	if plan.ID != "existing-plan" {
		t.Errorf("expected preserved plan ID, got %s", plan.ID)
	}

	// Should preserve original creation time
	if plan.CreatedAt.After(time.Now().Add(-30 * time.Minute)) {
		t.Error("expected preserved creation time")
	}
}

func TestPlanReconciler_ReconcileSkipsMalformed(t *testing.T) {
	reconciler := planning.NewPlanReconciler()

	proposed := []planning.Task{
		{ID: "", Title: "No ID"}, // Should be skipped
		{ID: "t1", Title: ""},    // Should be skipped
		{ID: "t2", Title: "Valid Task"},
	}

	plan, err := reconciler.Reconcile(nil, proposed, planning.ReconcileOptions{
		SpecID: "spec-1",
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Errorf("expected 1 valid task, got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].ID != "t2" {
		t.Errorf("expected t2, got %s", plan.Tasks[0].ID)
	}
}

func TestPlanReconciler_ReconcileDetectsCycles(t *testing.T) {
	reconciler := planning.NewPlanReconciler()

	// Create a cycle: t1 -> t2 -> t1
	proposed := []planning.Task{
		{ID: "t1", Title: "Task 1", DependsOn: []string{"t2"}},
		{ID: "t2", Title: "Task 2", DependsOn: []string{"t1"}},
	}

	_, err := reconciler.Reconcile(nil, proposed, planning.ReconcileOptions{
		SpecID: "spec-1",
	})
	if err == nil {
		t.Error("expected error for cyclic dependency")
	}
}

func TestPlanReconciler_FilterValidTasks(t *testing.T) {
	reconciler := planning.NewPlanReconciler()

	tasks := []planning.Task{
		{ID: "task-r1", Title: "Task for R1", FeatureID: "f1"},
		{ID: "task-r2", Title: "Task for R2", FeatureID: "f2"},
		{ID: "task-orphan", Title: "Orphan", FeatureID: "nonexistent"},
	}

	validTaskIDs := map[string]bool{"task-r1": true}
	validFeatureIDs := map[string]bool{"f1": true, "f2": true}

	filtered := reconciler.FilterValidTasks(tasks, validTaskIDs, validFeatureIDs)

	if len(filtered) != 2 {
		t.Errorf("expected 2 valid tasks, got %d", len(filtered))
	}

	// Orphan should be filtered out
	for _, task := range filtered {
		if task.ID == "task-orphan" {
			t.Error("orphan task should have been filtered out")
		}
	}
}

func TestPlanReconciler_ReconcileWithExplicitOptions(t *testing.T) {
	reconciler := planning.NewPlanReconciler()

	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	proposed := []planning.Task{
		{ID: "t1", Title: "Task 1"},
	}

	plan, err := reconciler.Reconcile(nil, proposed, planning.ReconcileOptions{
		SpecID:     "spec-1",
		ExistingID: "explicit-id",
		CreatedAt:  createdAt,
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if plan.ID != "explicit-id" {
		t.Errorf("expected explicit-id, got %s", plan.ID)
	}
	if !plan.CreatedAt.Equal(createdAt) {
		t.Errorf("expected explicit creation time")
	}
}
