package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	Stream   bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage,omitempty"`
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

	streaming := req.IsStreaming()

	body, err := json.Marshal(openAIRequest{
		Model:    p.Model,
		Messages: messages,
		Stream:   streaming,
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
	if streaming {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

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

	if streaming {
		return p.consumeSSE(ctx, resp, req.OnToken)
	}

	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, err
	}
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no choices")
	}
	return &ai.CompletionResponse{
		Text:       openAIResp.Choices[0].Message.Content,
		Model:      p.Model,
		Usage:      ai.TokenUsage{InputTokens: openAIResp.Usage.PromptTokens, OutputTokens: openAIResp.Usage.CompletionTokens},
		Confidence: confidenceFromStopReason(openAIResp.Choices[0].FinishReason),
	}, nil
}

func (p *OpenAIProvider) consumeSSE(ctx context.Context, resp *http.Response, onToken func(string)) (*ai.CompletionResponse, error) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var assembled strings.Builder
	var usage ai.TokenUsage
	var finishReason string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				assembled.WriteString(choice.Delta.Content)
				if onToken != nil {
					onToken(choice.Delta.Content)
				}
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
		if chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("openai stream read: %w", err)
	}
	if assembled.Len() == 0 {
		return nil, fmt.Errorf("openai stream returned no content")
	}

	return &ai.CompletionResponse{
		Text:       assembled.String(),
		Model:      p.Model,
		Usage:      usage,
		Confidence: confidenceFromStopReason(finishReason),
	}, nil
}
