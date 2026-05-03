// End-to-end streaming eval. Confirms the on-token callback installed by
// a CLI/MCP layer (via ai.WithOnToken) actually rides through the AI
// service into the provider and produces incremental chunks.
package evals

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/ai"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestStreamingFlowsCallerCallback(t *testing.T) {
	repo := setupAIRepo(t, &spec.ProductSpec{
		ID: "demo", Title: "Demo",
		Features: []spec.Feature{
			{ID: "f", Title: "F", Requirements: []spec.Requirement{{ID: "r", Title: "R"}}},
		},
	})

	// MockProvider streams JSON as a single chunk; that single chunk must
	// reach the caller-supplied callback when WithOnToken is installed.
	provider := &ai.MockProvider{Model: "stream"}

	var chunks []string
	ctx := domainai.WithOnToken(context.Background(), func(c string) {
		chunks = append(chunks, c)
	})

	if _, err := runDecomposeRawCtx(ctx, repo, provider); err != nil {
		t.Fatalf("streaming decompose: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected at least one streamed chunk to reach the caller")
	}
}
