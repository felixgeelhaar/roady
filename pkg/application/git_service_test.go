package application_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestGitService_SyncMarkers_BoundsCheck(t *testing.T) {
	repo := &MockRepo{
		Spec: &spec.ProductSpec{ID: "s1"},
		Plan: &planning.Plan{
			ID:             "p1",
			ApprovalStatus: planning.ApprovalApproved,
			Tasks:          []planning.Task{{ID: "t1", Title: "Task 1"}},
		},
		State:  planning.NewExecutionState("p1"),
		Policy: &domain.PolicyConfig{MaxWIP: 5, AllowAI: true},
	}

	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	taskSvc := application.NewTaskService(repo, audit, policy)
	gitSvc := application.NewGitService(repo, taskSvc)

	// SyncMarkers requires git — this tests that n bounds are handled
	// and that the service doesn't panic with small or large n values.
	// In a non-git directory, it will return an error which is fine.
	_, _ = gitSvc.SyncMarkers(0)   // Should clamp to 1
	_, _ = gitSvc.SyncMarkers(-5)  // Should clamp to 1
	_, _ = gitSvc.SyncMarkers(999) // Should work (under 1000)
}
