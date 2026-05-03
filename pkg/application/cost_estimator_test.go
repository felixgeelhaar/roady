package application

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestCostEstimator_KnownPricing(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := repo.SaveSpec(testSpec()); err != nil {
		t.Fatalf("save: %v", err)
	}

	cases := []struct {
		provider, model string
		wantPositive    bool
	}{
		{"anthropic", "claude-sonnet-4-6", true},
		{"openai", "gpt-4o-mini", true},
		{"gemini", "gemini-2.0-flash", true},
		{"ollama", "llama3", false},         // free / unknown
		{"unknown", "made-up-model", false}, // unknown
	}

	for _, tc := range cases {
		t.Run(tc.provider+"/"+tc.model, func(t *testing.T) {
			est := NewCostEstimator(repo, tc.provider, tc.model)
			out, err := est.Estimate("generate_plan")
			if err != nil {
				t.Fatalf("estimate: %v", err)
			}
			if out.InputTokensEstimate <= 0 {
				t.Errorf("expected positive input tokens, got %d", out.InputTokensEstimate)
			}
			if out.OutputTokensEstimate <= 0 {
				t.Errorf("expected positive output tokens, got %d", out.OutputTokensEstimate)
			}
			if tc.wantPositive {
				if !out.PricingKnown {
					t.Errorf("expected PricingKnown=true for %s/%s", tc.provider, tc.model)
				}
				if out.EstimatedCostUSD <= 0 {
					t.Errorf("expected positive cost for %s/%s, got %.6f", tc.provider, tc.model, out.EstimatedCostUSD)
				}
			} else {
				if out.PricingKnown {
					t.Errorf("expected PricingKnown=false for %s/%s", tc.provider, tc.model)
				}
				if out.EstimatedCostUSD != 0 {
					t.Errorf("expected zero cost for %s/%s, got %.6f", tc.provider, tc.model, out.EstimatedCostUSD)
				}
			}
		})
	}
}

func TestCostEstimator_DefaultsAndUnknownOperation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	_ = repo.Initialize()
	_ = repo.SaveSpec(testSpec())

	est := NewCostEstimator(repo, "anthropic", "claude-haiku-4-5")

	// Empty operation defaults to generate_plan.
	out, err := est.Estimate("")
	if err != nil {
		t.Fatalf("default op: %v", err)
	}
	if out.Operation != "generate_plan" {
		t.Errorf("default operation = %q, want generate_plan", out.Operation)
	}

	if _, err := est.Estimate("not_a_real_op"); err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("expected unknown-operation error, got %v", err)
	}
}

func TestCostEstimator_AllOperationsProduceEstimate(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	_ = repo.Initialize()
	_ = repo.SaveSpec(testSpec())
	est := NewCostEstimator(repo, "openai", "gpt-4o")

	for _, op := range []string{"generate_plan", "smart_decompose", "review_spec", "explain_drift", "query"} {
		t.Run(op, func(t *testing.T) {
			out, err := est.Estimate(op)
			if err != nil {
				t.Fatalf("%s: %v", op, err)
			}
			if out.InputTokensEstimate <= 0 || out.OutputTokensEstimate <= 0 {
				t.Errorf("%s: zero tokens %+v", op, out)
			}
		})
	}
}

func testSpec() *spec.ProductSpec {
	return &spec.ProductSpec{
		ID:          "demo",
		Title:       "Demo project",
		Description: "A reasonably-sized spec for cost estimation tests.",
		Version:     "0.1.0",
		Features: []spec.Feature{
			{
				ID:          "f1",
				Title:       "Feature One",
				Description: "Some description for the first feature so it's not trivially small.",
				Requirements: []spec.Requirement{
					{ID: "r1", Title: "Req One", Description: "Do the first thing.", Priority: "high"},
					{ID: "r2", Title: "Req Two", Description: "Do the second thing.", Priority: "medium", DependsOn: []string{"r1"}},
				},
			},
		},
	}
}
