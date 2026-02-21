package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/felixgeelhaar/roady/pkg/domain/ai"
)

type OpenAIProvider struct {
	Model      string
	APIKey     string
	baseURL    string       // For testing - defaults to OpenAI API
	httpClient *http.Client // For testing - defaults to http.DefaultClient
}

func NewOpenAIProvider(model string, apiKey string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIProvider{
		Model:   model,
		APIKey:  apiKey,
		baseURL: "https://api.openai.com/v1/chat/completions",
	}
}

// NewOpenAIProviderWithClient creates a provider with custom HTTP client and base URL (for testing).
func NewOpenAIProviderWithClient(model, apiKey, baseURL string, client *http.Client) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/chat/completions"
	}
	return &OpenAIProvider{
		Model:      model,
		APIKey:     apiKey,
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (p *OpenAIProvider) ID() string {
	return "openai:" + p.Model
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key not provided (set OPENAI_API_KEY)")
	}

	messages := []openAIMessage{}
	if req.System != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: req.System})
	}
	messages = append(messages, openAIMessage{Role: "user", Content: req.Prompt})

	body, err := json.Marshal(openAIRequest{
		Model:    p.Model,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := p.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read body

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status: %s", resp.Status)
	}

	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, err
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no choices")
	}

	return &ai.CompletionResponse{
		Text:  openAIResp.Choices[0].Message.Content,
		Model: p.Model,
		Usage: ai.TokenUsage{
			InputTokens:  openAIResp.Usage.PromptTokens,
			OutputTokens: openAIResp.Usage.CompletionTokens,
		},
	}, nil
}
