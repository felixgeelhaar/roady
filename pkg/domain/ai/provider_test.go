package ai

import (
	"context"
	"fmt"
	"testing"
)

// mockProvider implements the Provider interface for testing.
type mockProvider struct {
	id       string
	response *CompletionResponse
	err      error
}

func (m *mockProvider) ID() string { return m.id }
func (m *mockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestProvider_InterfaceContract(t *testing.T) {
	// Verify that mockProvider satisfies the Provider interface
	var _ Provider = &mockProvider{}
}

func TestProvider_Complete_Success(t *testing.T) {
	provider := &mockProvider{
		id: "test-provider",
		response: &CompletionResponse{
			Text:  "Hello, world!",
			Model: "test-model",
			Usage: TokenUsage{InputTokens: 10, OutputTokens: 5},
		},
	}

	if provider.ID() != "test-provider" {
		t.Errorf("ID() = %s, want test-provider", provider.ID())
	}

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Prompt:      "Say hello",
		System:      "You are a test assistant",
		Temperature: 0.7,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "Hello, world!" {
		t.Errorf("Text = %s, want Hello, world!", resp.Text)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d, want 5", resp.Usage.OutputTokens)
	}
}

func TestProvider_Complete_Error(t *testing.T) {
	provider := &mockProvider{
		id:  "error-provider",
		err: fmt.Errorf("connection refused"),
	}

	_, err := provider.Complete(context.Background(), CompletionRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "connection refused" {
		t.Errorf("error = %v, want connection refused", err)
	}
}

func TestProvider_Complete_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	provider := &mockProvider{
		id: "ctx-provider",
		response: &CompletionResponse{
			Text: "should not matter",
		},
	}

	// A well-behaved provider would check ctx.Err(), but our mock doesn't.
	// This test verifies the interface accepts a cancelled context.
	resp, err := provider.Complete(ctx, CompletionRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("mock doesn't check context: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
}

func TestCompletionRequest_Fields(t *testing.T) {
	req := CompletionRequest{
		Prompt:      "Generate tasks",
		System:      "You are a project planner",
		Temperature: 0.5,
		MaxTokens:   2000,
	}

	if req.Prompt != "Generate tasks" {
		t.Errorf("Prompt = %s", req.Prompt)
	}
	if req.System != "You are a project planner" {
		t.Errorf("System = %s", req.System)
	}
	if req.Temperature != 0.5 {
		t.Errorf("Temperature = %f", req.Temperature)
	}
	if req.MaxTokens != 2000 {
		t.Errorf("MaxTokens = %d", req.MaxTokens)
	}
}

func TestTokenUsage_Fields(t *testing.T) {
	usage := TokenUsage{InputTokens: 100, OutputTokens: 200}
	total := usage.InputTokens + usage.OutputTokens
	if total != 300 {
		t.Errorf("total tokens = %d, want 300", total)
	}
}

func TestCompletionResponse_HasConfidence(t *testing.T) {
	cases := []struct {
		name string
		resp *CompletionResponse
		want bool
	}{
		{"nil response", nil, false},
		{"zero confidence", &CompletionResponse{}, false},
		{"low confidence", &CompletionResponse{Confidence: 0.1}, true},
		{"high confidence", &CompletionResponse{Confidence: 0.92}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.resp.HasConfidence(); got != tc.want {
				t.Errorf("HasConfidence() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSourceRef_PreservesFields(t *testing.T) {
	ref := SourceRef{Doc: "docs/auth.md", Span: "lines 12-18", Note: "from MUST clause"}
	if ref.Doc != "docs/auth.md" || ref.Span != "lines 12-18" || ref.Note != "from MUST clause" {
		t.Errorf("SourceRef did not round-trip: %+v", ref)
	}
}

func TestWithOnToken_RoundTrip(t *testing.T) {
	ctx := context.Background()
	if OnTokenFromContext(ctx) != nil {
		t.Fatal("expected nil from empty context")
	}

	var got string
	withCb := WithOnToken(ctx, func(s string) { got = s })
	cb := OnTokenFromContext(withCb)
	if cb == nil {
		t.Fatal("expected callback in context")
	}
	cb("hello")
	if got != "hello" {
		t.Errorf("callback not invoked correctly, got=%q", got)
	}
}

func TestWithOnToken_NilFnReturnsParent(t *testing.T) {
	ctx := context.Background()
	out := WithOnToken(ctx, nil)
	if OnTokenFromContext(out) != nil {
		t.Error("nil callback should not install anything")
	}
}

func TestCompletionRequest_IsStreaming(t *testing.T) {
	if (CompletionRequest{}).IsStreaming() {
		t.Error("zero-value request should not be streaming")
	}
	streaming := CompletionRequest{OnToken: func(string) {}}
	if !streaming.IsStreaming() {
		t.Error("request with OnToken should be streaming")
	}
}

func TestCompletionResponse_OptionalFieldsBackwardCompatible(t *testing.T) {
	// Existing callers ignore the new fields; constructing without them
	// must still succeed and behave as "unknown".
	resp := &CompletionResponse{Text: "ok", Model: "x", Usage: TokenUsage{InputTokens: 1, OutputTokens: 2}}
	if resp.HasConfidence() {
		t.Error("expected HasConfidence false for legacy response")
	}
	if len(resp.Sources) != 0 {
		t.Errorf("expected nil Sources, got %v", resp.Sources)
	}
}
