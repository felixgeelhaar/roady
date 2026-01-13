package policy_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
)

type MockRule struct {
	Fail bool
}

func (m *MockRule) ID() string { return "mock" }
func (m *MockRule) Validate(plan *planning.Plan, state *planning.ExecutionState) []policy.Violation {
	if m.Fail {
		return []policy.Violation{{RuleID: "mock", Message: "failed"}}
	}
	return nil
}

func TestPolicySet_Validate(t *testing.T) {
	ps := policy.PolicySet{
		Rules: []policy.Rule{
			&MockRule{Fail: false},
			&MockRule{Fail: true},
		},
	}

	violations := ps.Validate(&planning.Plan{}, &planning.ExecutionState{})
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
	}
}
