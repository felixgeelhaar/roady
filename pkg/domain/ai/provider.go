package ai

import (
	"context"
)

// CompletionRequest represents a prompt to the AI.
type CompletionRequest struct {
	Prompt      string
	System      string
	Temperature float32
	MaxTokens   int
}

// CompletionResponse represents the AI's answer.
type CompletionResponse struct {
	Text  string
	Usage TokenUsage
	Model string
}

// TokenUsage tracks costs.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// Provider is the interface for all AI backends.
type Provider interface {
	ID() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}
