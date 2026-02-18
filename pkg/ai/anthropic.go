package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/felixgeelhaar/roady/pkg/domain/ai"
)

type AnthropicProvider struct {
	Model  string
	APIKey string
}

func NewAnthropicProvider(model string, apiKey string) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}
	return &AnthropicProvider{
		Model:  model,
		APIKey: apiKey,
	}
}

func (p *AnthropicProvider) ID() string {
	return "anthropic:" + p.Model
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key not provided (set ANTHROPIC_API_KEY)")
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	body, err := json.Marshal(anthropicRequest{
		Model:  p.Model,
		System: req.System,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.Prompt},
		},
		MaxTokens: maxTokens,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read body

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API returned status: %s", resp.Status)
	}

	var anthroResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthroResp); err != nil {
		return nil, err
	}

	if len(anthroResp.Content) == 0 {
		return nil, fmt.Errorf("Anthropic API returned no content")
	}

	return &ai.CompletionResponse{
		Text:  anthroResp.Content[0].Text,
		Model: p.Model,
		Usage: ai.TokenUsage{
			InputTokens:  anthroResp.Usage.InputTokens,
			OutputTokens: anthroResp.Usage.OutputTokens,
		},
	}, nil
}
