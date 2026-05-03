package application

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

// CostEstimate is the pre-flight projection an agent or human can inspect
// before triggering an AI operation.
type CostEstimate struct {
	Operation             string  `json:"operation"`
	Provider              string  `json:"provider"`
	Model                 string  `json:"model"`
	InputTokensEstimate   int     `json:"input_tokens_estimate"`
	OutputTokensEstimate  int     `json:"output_tokens_estimate"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd"`
	PricingKnown          bool    `json:"pricing_known"`
	Notes                 string  `json:"notes,omitempty"`
}

// modelPrice tracks USD per 1,000,000 tokens for a model. Numbers are
// best-effort published list prices and should be reviewed periodically.
type modelPrice struct {
	InputPerMTokens  float64
	OutputPerMTokens float64
}

// modelPricing is keyed by lower-cased model identifier substring so that
// minor model name variants (e.g. snapshot suffixes) match the right tier.
var modelPricing = []struct {
	match string
	price modelPrice
}{
	// Anthropic — published 2025/2026 list prices.
	{"claude-haiku", modelPrice{InputPerMTokens: 1.0, OutputPerMTokens: 5.0}},
	{"claude-sonnet", modelPrice{InputPerMTokens: 3.0, OutputPerMTokens: 15.0}},
	{"claude-opus", modelPrice{InputPerMTokens: 15.0, OutputPerMTokens: 75.0}},
	// OpenAI.
	{"gpt-4o-mini", modelPrice{InputPerMTokens: 0.15, OutputPerMTokens: 0.60}},
	{"gpt-4o", modelPrice{InputPerMTokens: 5.0, OutputPerMTokens: 20.0}},
	{"gpt-5", modelPrice{InputPerMTokens: 10.0, OutputPerMTokens: 30.0}},
	{"gpt-4", modelPrice{InputPerMTokens: 30.0, OutputPerMTokens: 60.0}},
	// Google Gemini.
	{"gemini-2", modelPrice{InputPerMTokens: 0.10, OutputPerMTokens: 0.40}},
	{"gemini-1.5-pro", modelPrice{InputPerMTokens: 3.5, OutputPerMTokens: 10.5}},
	{"gemini-1.5-flash", modelPrice{InputPerMTokens: 0.075, OutputPerMTokens: 0.30}},
}

// lookupPrice returns the pricing tier for a model and whether it was found.
// Ollama or any unknown model returns (zero, false), which the caller treats
// as free / unknown rather than zero-cost.
func lookupPrice(provider, model string) (modelPrice, bool) {
	if strings.EqualFold(provider, "ollama") || strings.EqualFold(provider, "mock") {
		return modelPrice{}, false
	}
	m := strings.ToLower(model)
	for _, entry := range modelPricing {
		if strings.Contains(m, entry.match) {
			return entry.price, true
		}
	}
	return modelPrice{}, false
}

// CostEstimator computes pre-flight token + USD projections for AI ops. It
// reads from the workspace repository so callers do not need to plumb
// service references; pricing is derived from the configured provider/model.
type CostEstimator struct {
	repo     domain.WorkspaceRepository
	provider string
	model    string
}

// NewCostEstimator builds an estimator for the supplied repository and AI
// provider/model identifiers (typically the values resolved by wiring).
func NewCostEstimator(repo domain.WorkspaceRepository, provider, model string) *CostEstimator {
	return &CostEstimator{repo: repo, provider: provider, model: model}
}

// Estimate returns a projection for one of the supported AI operations.
// Unknown operation names produce an error so agents fail fast rather than
// silently relying on a wrong default.
func (c *CostEstimator) Estimate(operation string) (*CostEstimate, error) {
	op := strings.ToLower(strings.TrimSpace(operation))
	if op == "" {
		op = "generate_plan"
	}

	productSpec, _ := c.repo.LoadSpec()
	specChars := specCharCount(productSpec)
	specTokens := tokensFromChars(specChars)

	in, out, notes, err := tokenProjectionFor(op, productSpec, specTokens)
	if err != nil {
		return nil, err
	}

	price, known := lookupPrice(c.provider, c.model)
	cost := 0.0
	if known {
		cost = (float64(in)/1_000_000)*price.InputPerMTokens +
			(float64(out)/1_000_000)*price.OutputPerMTokens
	}

	return &CostEstimate{
		Operation:            op,
		Provider:             c.provider,
		Model:                c.model,
		InputTokensEstimate:  in,
		OutputTokensEstimate: out,
		EstimatedCostUSD:     cost,
		PricingKnown:         known,
		Notes:                notes,
	}, nil
}

// tokensFromChars converts a character count to an approximate token count
// using the well-known ~4-chars-per-token rule of thumb. Good enough for
// pre-flight estimates; real-world drift from the actual tokenizer is
// typically within 10-20%.
func tokensFromChars(chars int) int {
	if chars <= 0 {
		return 0
	}
	tokens := chars / 4
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

func specCharCount(s *spec.ProductSpec) int {
	if s == nil {
		return 0
	}
	total := len(s.ID) + len(s.Title) + len(s.Description)
	for _, f := range s.Features {
		total += len(f.ID) + len(f.Title) + len(f.Description)
		for _, r := range f.Requirements {
			total += len(r.ID) + len(r.Title) + len(r.Description) + len(r.Estimate) + len(r.Priority)
		}
	}
	for _, c := range s.Constraints {
		total += len(c.ID) + len(c.Description)
	}
	return total
}

// tokenProjectionFor maps an operation to (input, output, notes). Heuristic
// ratios are tuned against the existing AI prompts in pkg/application; if a
// prompt grows materially these numbers should be revisited.
func tokenProjectionFor(op string, productSpec *spec.ProductSpec, specTokens int) (int, int, string, error) {
	requirementCount := 0
	featureCount := 0
	if productSpec != nil {
		featureCount = len(productSpec.Features)
		for _, f := range productSpec.Features {
			requirementCount += len(f.Requirements)
		}
	}

	const systemPromptTokens = 600

	switch op {
	case "generate_plan":
		in := specTokens + systemPromptTokens
		out := 200 + 80*requirementCount
		if requirementCount == 0 {
			out = 200 + 80*featureCount
		}
		return in, out, "Output scales with requirement count (~80 tokens/task).", nil

	case "smart_decompose":
		// Adds a codebase summary on top of the spec.
		const codebaseSummaryTokens = 4000
		in := specTokens + codebaseSummaryTokens + systemPromptTokens
		out := 400 + 120*featureCount
		return in, out, "Includes a codebase summary in the prompt; output scales with feature count.", nil

	case "review_spec":
		in := specTokens + systemPromptTokens
		out := 1200
		return in, out, "Output is a fixed-size completeness review.", nil

	case "explain_drift":
		// Drift reports are small; bound input.
		const driftReportTokens = 1500
		in := driftReportTokens + systemPromptTokens
		out := 800
		return in, out, "Drift reports are typically small; estimate is provider-agnostic.", nil

	case "query":
		// Whole project context is roughly spec + plan + state summary.
		in := specTokens + 2000 + systemPromptTokens
		out := 500
		return in, out, "Adds plan + state summary to the prompt.", nil

	default:
		return 0, 0, "", fmt.Errorf("unknown operation %q (supported: generate_plan, smart_decompose, review_spec, explain_drift, query)", op)
	}
}
