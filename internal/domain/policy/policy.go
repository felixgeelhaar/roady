package policy

import (
	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

type ViolationLevel string

const (
	ViolationWarning ViolationLevel = "warning"
	ViolationError   ViolationLevel = "error"
)

// Violation represents a breach of a policy rule.
type Violation struct {
	RuleID  string         `json:"rule_id"`
	Message string         `json:"message"`
	Level   ViolationLevel `json:"level"`
}

// Rule defines a constraint that can be validated against a plan and state.
type Rule interface {
	ID() string
	Validate(plan *planning.Plan, state *planning.ExecutionState) []Violation
}

// PolicySet is a collection of rules enabled for a project.
type PolicySet struct {
	Rules []Rule
}

func (ps *PolicySet) Validate(plan *planning.Plan, state *planning.ExecutionState) []Violation {
	var violations []Violation
	for _, rule := range ps.Rules {
		violations = append(violations, rule.Validate(plan, state)...)
	}
	return violations
}
