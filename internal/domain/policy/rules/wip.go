package rules

import (
	"fmt"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/domain/policy"
)

type MaxWIPRule struct {
	Limit int `yaml:"limit"`
}

func (r *MaxWIPRule) ID() string {
	return "max-wip"
}

func (r *MaxWIPRule) Validate(plan *planning.Plan, state *planning.ExecutionState) []policy.Violation {
	if plan == nil || state == nil {
		return nil
	}
	inProgressCount := 0
	for _, task := range plan.Tasks {
		if res, ok := state.TaskStates[task.ID]; ok {
			if res.Status == planning.StatusInProgress {
				inProgressCount++
			}
		}
	}

	if inProgressCount > r.Limit {
		return []policy.Violation{
			{
				RuleID:  r.ID(),
				Level:   policy.ViolationWarning,
				Message: fmt.Sprintf("WIP Limit Exceeded: %d tasks in progress (limit: %d).", inProgressCount, r.Limit),
			},
		}
	}

	return nil
}