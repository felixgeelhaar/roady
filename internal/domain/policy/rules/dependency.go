package rules

import (
	"fmt"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/domain/policy"
)

type DependencyRule struct{}

func (r *DependencyRule) ID() string {
	return "dependency-check"
}

func (r *DependencyRule) Validate(plan *planning.Plan, state *planning.ExecutionState) []policy.Violation {
	if plan == nil || state == nil {
		return nil
	}
	var violations []policy.Violation

	// Create a map for quick status lookup
	statusMap := make(map[string]planning.TaskStatus)
	for id, res := range state.TaskStates {
		statusMap[id] = res.Status
	}

	for _, task := range plan.Tasks {
		// Only check tasks that are In Progress
		if statusMap[task.ID] != planning.StatusInProgress {
			continue
		}

		for _, depID := range task.DependsOn {
			if statusMap[depID] != planning.StatusDone {
				violations = append(violations, policy.Violation{
					RuleID:  r.ID(),
					Level:   policy.ViolationError,
					Message: fmt.Sprintf("Task '%s' is in progress but depends on '%s' which is not done.", task.ID, depID),
				})
			}
		}
	}

	return violations
}
