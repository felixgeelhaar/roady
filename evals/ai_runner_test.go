// AI planner eval. Drives DecomposeSpec with a programmable mock provider
// that returns canned JSON shaped like a real model output. Locks the AI
// pipeline contract: parsing, origin tagging, feature coverage, and
// fallback-task behaviour.
//
// Real-provider runs are intentionally not in CI to keep the suite free.
// A future EVAL_AI=1 build tag can enable them locally.
package evals

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// programmableMock returns the same canned text for every Complete call.
// Tests configure it per-fixture so we can drive the real AI parser through
// happy-path and edge-case shapes.
type programmableMock struct {
	id        string
	respText  string
	callCount int
}

func (p *programmableMock) ID() string { return p.id }

func (p *programmableMock) Complete(_ context.Context, _ domainai.CompletionRequest) (*domainai.CompletionResponse, error) {
	p.callCount++
	return &domainai.CompletionResponse{
		Text:  p.respText,
		Model: "programmable-mock",
		Usage: domainai.TokenUsage{InputTokens: 50, OutputTokens: 200},
	}, nil
}

func TestAIDecomposeSpec_HappyPath(t *testing.T) {
	repo := setupAIRepo(t, &spec.ProductSpec{
		ID:    "demo",
		Title: "Demo",
		Features: []spec.Feature{
			{ID: "auth", Title: "Auth", Requirements: []spec.Requirement{
				{ID: "auth-signup", Title: "Sign up"},
				{ID: "auth-login", Title: "Log in"},
			}},
			{ID: "tasks", Title: "Tasks", Requirements: []spec.Requirement{
				{ID: "tasks-create", Title: "Create"},
			}},
		},
	})

	provider := &programmableMock{
		id: "mock:test",
		respText: `[
{"id":"task-auth-signup","title":"Sign up","feature_id":"auth","priority":"high","estimate":"1h"},
{"id":"task-auth-login","title":"Log in","feature_id":"auth","priority":"high","estimate":"1h"},
{"id":"task-tasks-create","title":"Create task","feature_id":"tasks","priority":"medium","estimate":"2h"}
]`,
	}

	plan := runDecompose(t, repo, provider)

	wantIDs := []string{"task-auth-login", "task-auth-signup", "task-tasks-create"}
	gotIDs := taskIDs(plan)
	if !equalSorted(gotIDs, wantIDs) {
		t.Errorf("task IDs = %v, want %v", gotIDs, wantIDs)
	}

	for _, task := range plan.Tasks {
		if task.NormalisedOrigin() != planning.OriginAI {
			t.Errorf("task %q origin = %q, want ai", task.ID, task.NormalisedOrigin())
		}
		if task.FeatureID == "" {
			t.Errorf("task %q missing feature_id", task.ID)
		}
	}
}

func TestAIDecomposeSpec_FillsMissingFeatureCoverage(t *testing.T) {
	// Spec has two features but the AI only returns a task for one. The
	// service should backfill a heuristic task for the missing feature.
	repo := setupAIRepo(t, &spec.ProductSpec{
		ID:    "demo",
		Title: "Demo",
		Features: []spec.Feature{
			{ID: "covered", Title: "Covered", Requirements: []spec.Requirement{{ID: "covered-r1", Title: "R"}}},
			{ID: "missing", Title: "Missing"},
		},
	})

	provider := &programmableMock{
		id:       "mock:test",
		respText: `[{"id":"task-covered-r1","title":"Do thing","feature_id":"covered","priority":"high"}]`,
	}

	plan := runDecompose(t, repo, provider)

	var sawMissing bool
	for _, task := range plan.Tasks {
		if task.FeatureID == "missing" {
			sawMissing = true
			if task.NormalisedOrigin() != planning.OriginHeuristic {
				t.Errorf("backfill task origin = %q, want heuristic", task.NormalisedOrigin())
			}
		}
	}
	if !sawMissing {
		t.Error("expected a backfill task for the uncovered feature")
	}
}

func TestAIDecomposeSpec_BackfillsSourceFromSpec(t *testing.T) {
	repo := setupAIRepo(t, &spec.ProductSpec{
		ID:    "demo",
		Title: "Demo",
		Features: []spec.Feature{
			{
				ID:     "auth",
				Title:  "Auth",
				Source: spec.Source{Doc: "docs/auth.md", Line: 7},
				Requirements: []spec.Requirement{
					{ID: "auth-signup", Title: "Sign up", Source: spec.Source{Doc: "docs/auth.md", Line: 14}},
				},
			},
		},
	})

	provider := &programmableMock{
		id:       "mock:test",
		respText: `[{"id":"task-auth-signup","title":"Sign up","feature_id":"auth","priority":"high"}]`,
	}

	plan := runDecompose(t, repo, provider)
	if len(plan.Tasks) == 0 {
		t.Fatal("expected at least one task")
	}

	got := plan.Tasks[0]
	want := planning.TaskSource{Doc: "docs/auth.md", Line: 14}
	if got.Source != want {
		t.Errorf("AI task lost source citation: got %+v, want %+v", got.Source, want)
	}
}

func TestAIDecomposeSpec_RespectsPolicyDisable(t *testing.T) {
	repo := setupAIRepo(t, &spec.ProductSpec{ID: "demo", Title: "Demo", Features: []spec.Feature{{ID: "f", Title: "F"}}})

	// Override policy to deny AI.
	if err := repo.SavePolicy(&policy.PolicyConfig{AllowAI: false, MaxWIP: 5}); err != nil {
		t.Fatal(err)
	}

	provider := &programmableMock{id: "mock:test", respText: "[]"}
	_, err := runDecomposeRaw(repo, provider)
	if err == nil || !strings.Contains(err.Error(), "AI usage is disabled") {
		t.Fatalf("expected policy-denied error, got %v", err)
	}
	if provider.callCount != 0 {
		t.Errorf("expected 0 provider calls when AI disabled, got %d", provider.callCount)
	}
}

// --- helpers ---

func setupAIRepo(t *testing.T, sp *spec.ProductSpec) *storage.FilesystemRepository {
	t.Helper()
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	if err := repo.SaveSpec(sp); err != nil {
		t.Fatalf("save spec: %v", err)
	}
	if err := repo.SavePolicy(&policy.PolicyConfig{AllowAI: true, MaxWIP: 5}); err != nil {
		t.Fatalf("save policy: %v", err)
	}
	return repo
}

func runDecompose(t *testing.T, repo *storage.FilesystemRepository, provider domainai.Provider) *planning.Plan {
	t.Helper()
	plan, err := runDecomposeRaw(repo, provider)
	if err != nil {
		t.Fatalf("decompose: %v", err)
	}
	if plan == nil {
		t.Fatal("nil plan")
	}
	return plan
}

func runDecomposeRaw(repo *storage.FilesystemRepository, provider domainai.Provider) (*planning.Plan, error) {
	return runDecomposeRawCtx(context.Background(), repo, provider)
}

func runDecomposeRawCtx(ctx context.Context, repo *storage.FilesystemRepository, provider domainai.Provider) (*planning.Plan, error) {
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	aiSvc := application.NewAIPlanningService(repo, provider, audit, planSvc)
	plan, err := aiSvc.DecomposeSpec(ctx)
	if err != nil {
		return nil, fmt.Errorf("DecomposeSpec: %w", err)
	}
	return plan, nil
}

func taskIDs(plan *planning.Plan) []string {
	out := make([]string, 0, len(plan.Tasks))
	for _, t := range plan.Tasks {
		out = append(out, t.ID)
	}
	sort.Strings(out)
	return out
}

func equalSorted(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
