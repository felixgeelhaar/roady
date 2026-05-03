// Drift detector precision/recall corpus.
//
// Each scenario stages a spec/plan/state trio plus an optional spec lock
// and policy, then asserts the DriftService surfaces exactly the expected
// (type, category) pairs. The healthy baseline asserts an empty report —
// our precision floor — while the divergence scenarios pin recall.
package evals

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/drift"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

type expectedIssue struct {
	Type     drift.DriftType
	Category drift.DriftCategory
}

func TestDriftCorpus(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(t *testing.T) *storage.FilesystemRepository
		expected []expectedIssue
	}{
		{
			name:     "healthy_baseline_no_drift",
			setup:    setupHealthyProject,
			expected: nil,
		},
		{
			name:  "intent_drift_when_spec_diverges_from_lock",
			setup: setupIntentDrift,
			expected: []expectedIssue{
				{Type: drift.DriftTypeSpec, Category: drift.CategoryMismatch},
			},
		},
		{
			name:  "plan_drift_when_requirement_missing_from_plan",
			setup: setupPlanDrift,
			expected: []expectedIssue{
				{Type: drift.DriftTypePlan, Category: drift.CategoryMissing},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.setup(t)
			report := runDriftDetect(t, repo)
			assertExactIssues(t, report, tc.expected)
		})
	}
}

// --- scenarios ---

func setupHealthyProject(t *testing.T) *storage.FilesystemRepository {
	t.Helper()
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init: %v", err)
	}

	sp := &spec.ProductSpec{
		ID:    "healthy",
		Title: "Healthy",
		Features: []spec.Feature{{
			ID:    "f1",
			Title: "Feature One",
			Requirements: []spec.Requirement{
				{ID: "r1", Title: "Requirement One"},
			},
		}},
	}
	if err := repo.SaveSpec(sp); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveSpecLock(sp); err != nil {
		t.Fatal(err)
	}

	plan := &planning.Plan{
		ID:             "plan-healthy",
		SpecID:         sp.ID,
		ApprovalStatus: planning.ApprovalApproved,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Tasks: []planning.Task{
			{ID: "task-r1", FeatureID: "f1", Title: "Implement R1", Origin: planning.OriginHeuristic},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveState(planning.NewExecutionState(sp.ID)); err != nil {
		t.Fatal(err)
	}
	if err := repo.SavePolicy(&policy.PolicyConfig{MaxWIP: 5, AllowAI: true}); err != nil {
		t.Fatal(err)
	}
	return repo
}

func setupIntentDrift(t *testing.T) *storage.FilesystemRepository {
	t.Helper()
	repo := setupHealthyProject(t)

	// Mutate the spec without re-locking. The lock continues to point at
	// the original hash, so the detector should report intent drift.
	updated, _ := repo.LoadSpec()
	updated.Features = append(updated.Features, spec.Feature{
		ID:    "f2",
		Title: "Added After Lock",
		Requirements: []spec.Requirement{
			{ID: "r2", Title: "Drifted Requirement"},
		},
	})
	if err := repo.SaveSpec(updated); err != nil {
		t.Fatal(err)
	}

	// Add a corresponding plan task so the new requirement is covered;
	// this isolates the test to *intent* drift only.
	plan, _ := repo.LoadPlan()
	plan.Tasks = append(plan.Tasks, planning.Task{
		ID:        "task-r2",
		FeatureID: "f2",
		Title:     "Implement R2",
		Origin:    planning.OriginHeuristic,
	})
	if err := repo.SavePlan(plan); err != nil {
		t.Fatal(err)
	}
	return repo
}

func setupPlanDrift(t *testing.T) *storage.FilesystemRepository {
	t.Helper()
	repo := setupHealthyProject(t)

	// Add a requirement to the spec AND re-lock so intent drift stays
	// quiet. Leave the plan unchanged so the new requirement has no task.
	updated, _ := repo.LoadSpec()
	updated.Features[0].Requirements = append(updated.Features[0].Requirements, spec.Requirement{
		ID:    "r-uncovered",
		Title: "Uncovered Requirement",
	})
	if err := repo.SaveSpec(updated); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveSpecLock(updated); err != nil {
		t.Fatal(err)
	}
	return repo
}

// --- helpers ---

func runDriftDetect(t *testing.T, repo *storage.FilesystemRepository) *drift.Report {
	t.Helper()
	audit := application.NewAuditService(repo)
	policySvc := application.NewPolicyService(repo)
	driftSvc := application.NewDriftService(repo, audit, storage.NewCodebaseInspector(), policySvc)

	report, err := driftSvc.DetectDrift(context.Background())
	if err != nil {
		t.Fatalf("DetectDrift: %v", err)
	}
	if report == nil {
		t.Fatal("nil report")
	}
	return report
}

func assertExactIssues(t *testing.T, report *drift.Report, expected []expectedIssue) {
	t.Helper()

	// Filter out implementation drift because evals run in a temp dir with
	// no codebase to inspect; the scenarios under test never exercise that
	// branch on purpose.
	var got []expectedIssue
	for _, issue := range report.Issues {
		if issue.Type == drift.DriftTypeCode {
			continue
		}
		got = append(got, expectedIssue{Type: issue.Type, Category: issue.Category})
	}

	sortIssues(got)
	wantCopy := append([]expectedIssue(nil), expected...)
	sortIssues(wantCopy)

	if !sameIssues(got, wantCopy) {
		t.Fatalf("drift issues mismatch\n  got:  %v\n  want: %v\n  full: %s",
			got, wantCopy, dumpReport(report))
	}
}

func sortIssues(in []expectedIssue) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Type != in[j].Type {
			return in[i].Type < in[j].Type
		}
		return in[i].Category < in[j].Category
	})
}

func sameIssues(a, b []expectedIssue) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func dumpReport(report *drift.Report) string {
	var b strings.Builder
	for _, issue := range report.Issues {
		b.WriteString(string(issue.Type))
		b.WriteString("/")
		b.WriteString(string(issue.Category))
		b.WriteString(": ")
		b.WriteString(issue.Message)
		b.WriteString("\n")
	}
	return b.String()
}
