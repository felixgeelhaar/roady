//go:build evals_ai

// Real-provider eval matrix. Off by default; opt in with:
//
//	go test -tags evals_ai ./evals/...
//
// Requires the corresponding provider's API key in the environment.
// Providers without keys configured are skipped (not failed) so the suite
// runs cleanly with whatever keys the operator has on hand. Each scenario
// sends a small spec, asks for a JSON plan, and verifies the AI service
// returns a non-empty plan with each task tagged Origin=ai. Streaming is
// exercised via ai.WithOnToken.
package evals

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/ai"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func realProviderSpec() *spec.ProductSpec {
	return &spec.ProductSpec{
		ID: "real-eval", Title: "Real Provider Eval",
		Features: []spec.Feature{
			{ID: "auth", Title: "Auth", Requirements: []spec.Requirement{
				{ID: "auth-signup", Title: "Sign up"},
				{ID: "auth-login", Title: "Log in"},
			}},
		},
	}
}

type providerCase struct {
	name    string
	envKey  string
	build   func(model, key string) domainai.Provider
	model   string
}

func providerCases() []providerCase {
	return []providerCase{
		{"anthropic", "ANTHROPIC_API_KEY", func(m, k string) domainai.Provider { return ai.NewAnthropicProvider(m, k) }, ""},
		{"openai", "OPENAI_API_KEY", func(m, k string) domainai.Provider { return ai.NewOpenAIProvider(m, k) }, ""},
		{"gemini", "GEMINI_API_KEY", func(m, k string) domainai.Provider { return ai.NewGeminiProvider(m, k) }, ""},
		{"ollama", "ROADY_EVAL_ENABLE_OLLAMA", func(m, k string) domainai.Provider { return ai.NewOllamaProvider(m) }, ""},
	}
}

func TestRealProviders_DecomposeStreams(t *testing.T) {
	for _, pc := range providerCases() {
		pc := pc
		t.Run(pc.name, func(t *testing.T) {
			key := os.Getenv(pc.envKey)
			if key == "" {
				t.Skipf("%s not set; skipping", pc.envKey)
			}
			provider := pc.build(pc.model, key)
			repo := setupAIRepo(t, realProviderSpec())

			var chunkCount atomic.Int64
			ctx := domainai.WithOnToken(context.Background(), func(c string) {
				if strings.TrimSpace(c) != "" {
					chunkCount.Add(1)
				}
			})

			plan, err := runDecomposeRawCtx(ctx, repo, provider)
			if err != nil {
				t.Fatalf("DecomposeSpec via %s: %v", pc.name, err)
			}
			if len(plan.Tasks) == 0 {
				t.Fatalf("%s returned empty plan", pc.name)
			}
			for _, task := range plan.Tasks {
				if task.NormalisedOrigin() != planning.OriginAI && task.NormalisedOrigin() != planning.OriginHeuristic {
					t.Errorf("%s task %q origin = %q, expected ai or heuristic (backfill)", pc.name, task.ID, task.NormalisedOrigin())
				}
			}
			// Streaming-capable providers (everyone in this matrix) MUST
			// emit at least one chunk. If they do not, the OnToken sink
			// never fired and we are silently regressing.
			if chunkCount.Load() == 0 {
				t.Errorf("%s did not stream any chunks via OnToken", pc.name)
			}
		})
	}
}
