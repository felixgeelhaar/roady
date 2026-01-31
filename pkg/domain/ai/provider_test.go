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
