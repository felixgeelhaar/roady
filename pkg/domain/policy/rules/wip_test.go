package rules_test

import (
	"fmt"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/felixgeelhaar/roady/pkg/domain/policy/rules"
)

func TestMaxWIPRule_Validate(t *testing.T) {
	rule := &rules.MaxWIPRule{Limit: 2}

	tests := []struct {
		name       string
		statuses   []planning.TaskStatus
		wantIssues int
	}{
		{
			name:       "Below Limit",
			statuses:   []planning.TaskStatus{planning.StatusInProgress, planning.StatusDone},
			wantIssues: 0,
		},
		{
			name:       "At Limit",
			statuses:   []planning.TaskStatus{planning.StatusInProgress, planning.StatusInProgress},
			wantIssues: 0,
		},
		{
			name:       "Above Limit",
			statuses:   []planning.TaskStatus{planning.StatusInProgress, planning.StatusInProgress, planning.StatusInProgress},
			wantIssues: 1,
		},
		{
			name:       "Mixed States",
			statuses:   []planning.TaskStatus{planning.StatusInProgress, planning.StatusPending, planning.StatusBlocked, planning.StatusDone},
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planning.Plan{Tasks: []planning.Task{}}
			state := &planning.ExecutionState{
				TaskStates: make(map[string]planning.TaskResult),
			}

			for i, status := range tt.statuses {
				id := fmt.Sprintf("t%d", i)
				plan.Tasks = append(plan.Tasks, planning.Task{ID: id})
				state.TaskStates[id] = planning.TaskResult{Status: status}
			}

			violations := rule.Validate(plan, state)

			if len(violations) != tt.wantIssues {
				t.Errorf("got %d violations, want %d", len(violations), tt.wantIssues)
			}
			if len(violations) > 0 && violations[0].Level != policy.ViolationWarning {
				t.Errorf("expected warning violation, got %s", violations[0].Level)
			}
		})
	}
}