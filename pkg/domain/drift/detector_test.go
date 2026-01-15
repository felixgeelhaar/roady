package drift_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/drift"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestDriftDetector_DetectIntentDrift(t *testing.T) {
	detector := drift.NewDriftDetector()

	// No lock - no drift
	issues := detector.DetectIntentDrift(&spec.ProductSpec{ID: "test"}, nil)
	if len(issues) != 0 {
		t.Errorf("expected no issues when lock is nil, got %d", len(issues))
	}

	// Same spec - no drift
	s := &spec.ProductSpec{ID: "test", Title: "Test"}
	issues = detector.DetectIntentDrift(s, s)
	if len(issues) != 0 {
		t.Errorf("expected no issues when specs match, got %d", len(issues))
	}

	// Different spec - drift
	current := &spec.ProductSpec{ID: "test", Title: "Test v2"}
	locked := &spec.ProductSpec{ID: "test", Title: "Test v1"}
	issues = detector.DetectIntentDrift(current, locked)
	if len(issues) != 1 {
		t.Errorf("expected 1 intent drift issue, got %d", len(issues))
	}
	if issues[0].Type != drift.DriftTypeSpec {
		t.Errorf("expected spec drift type, got %s", issues[0].Type)
	}
}

func TestDriftDetector_DetectPlanDrift(t *testing.T) {
	detector := drift.NewDriftDetector()

	// Spec with requirement but no task in plan
	s := &spec.ProductSpec{
		ID: "test",
		Features: []spec.Feature{
			{
				ID:    "f1",
				Title: "Feature 1",
				Requirements: []spec.Requirement{
					{ID: "r1", Title: "Req 1"},
				},
			},
		},
	}
	plan := &planning.Plan{
		Tasks: []planning.Task{},
	}

	issues := detector.DetectPlanDrift(s, plan)
	if len(issues) != 1 {
		t.Errorf("expected 1 missing task issue, got %d", len(issues))
	}
	if issues[0].Category != drift.CategoryMissing {
		t.Errorf("expected missing category, got %s", issues[0].Category)
	}

	// Orphan task (in plan but not in spec)
	plan2 := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "task-orphan", Title: "Orphan", FeatureID: "nonexistent"},
		},
	}
	s2 := &spec.ProductSpec{ID: "test", Features: []spec.Feature{}}

	issues = detector.DetectPlanDrift(s2, plan2)
	if len(issues) != 1 {
		t.Errorf("expected 1 orphan task issue, got %d", len(issues))
	}
	if issues[0].Category != drift.CategoryOrphan {
		t.Errorf("expected orphan category, got %s", issues[0].Category)
	}
}

type mockInspector struct {
	exists   bool
	notEmpty bool
	status   string
}

func (m *mockInspector) FileExists(path string) (bool, error)   { return m.exists, nil }
func (m *mockInspector) FileNotEmpty(path string) (bool, error) { return m.notEmpty, nil }
func (m *mockInspector) GitStatus(path string) (string, error)  { return m.status, nil }

func TestDriftDetector_DetectCodeDrift(t *testing.T) {
	detector := drift.NewDriftDetector()

	plan := &planning.Plan{
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
		},
	}
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"t1": {Status: planning.StatusDone, Path: "/some/path.go"},
		},
	}

	// File missing
	inspector := &mockInspector{exists: false}
	issues := detector.DetectCodeDrift(plan, state, inspector)
	if len(issues) != 1 {
		t.Errorf("expected 1 missing file issue, got %d", len(issues))
	}
	if issues[0].Severity != drift.SeverityCritical {
		t.Errorf("expected critical severity, got %s", issues[0].Severity)
	}

	// File empty
	inspector = &mockInspector{exists: true, notEmpty: false}
	issues = detector.DetectCodeDrift(plan, state, inspector)
	if len(issues) != 1 {
		t.Errorf("expected 1 empty file issue, got %d", len(issues))
	}

	// File modified/uncommitted
	inspector = &mockInspector{exists: true, notEmpty: true, status: "modified"}
	issues = detector.DetectCodeDrift(plan, state, inspector)
	if len(issues) != 1 {
		t.Errorf("expected 1 uncommitted issue, got %d", len(issues))
	}

	// Clean file - no issues
	inspector = &mockInspector{exists: true, notEmpty: true, status: "clean"}
	issues = detector.DetectCodeDrift(plan, state, inspector)
	if len(issues) != 0 {
		t.Errorf("expected no issues for clean file, got %d", len(issues))
	}
}

func TestDriftDetector_DetectPolicyDrift(t *testing.T) {
	detector := drift.NewDriftDetector()

	violations := []policy.Violation{
		{RuleID: "max-wip", Level: "warning", Message: "WIP exceeded"},
		{RuleID: "blocked", Level: "error", Message: "Blocked tasks"},
	}

	issues := detector.DetectPolicyDrift(violations)
	if len(issues) != 2 {
		t.Errorf("expected 2 policy issues, got %d", len(issues))
	}

	// Warning -> medium severity
	if issues[0].Severity != drift.SeverityMedium {
		t.Errorf("expected medium severity for warning, got %s", issues[0].Severity)
	}

	// Error -> high severity
	if issues[1].Severity != drift.SeverityHigh {
		t.Errorf("expected high severity for error, got %s", issues[1].Severity)
	}
}
