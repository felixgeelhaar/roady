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

type AnthropicProvider struct {
	Model      string
	APIKey     string
	baseURL    string       // overridable for tests; "" => production endpoint
	httpClient *http.Client // overridable for tests; nil => http.DefaultClient
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

// NewAnthropicProviderWithClient lets tests inject an httptest server.
func NewAnthropicProviderWithClient(model, apiKey, baseURL string, client *http.Client) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}
	return &AnthropicProvider{
		Model:      model,
		APIKey:     apiKey,
		baseURL:    baseURL,
		httpClient: client,
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
	Stream    bool               `json:"stream,omitempty"`
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
	StopReason string `json:"stop_reason"`
}

// Anthropic SSE event payloads we care about. Other event types are
// ignored; the message_delta carries final usage updates.
type anthropicSSEDelta struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type       string `json:"type"`
		Text       string `json:"text"`
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Message struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *AnthropicProvider) endpoint() string {
	if p.baseURL != "" {
		return p.baseURL
	}
	return "https://api.anthropic.com/v1/messages"
}

func (p *AnthropicProvider) client() *http.Client {
	if p.httpClient != nil {
		return p.httpClient
	}
	return http.DefaultClient
}

func (p *AnthropicProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key not provided (set ANTHROPIC_API_KEY)")
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	streaming := req.IsStreaming()

	body, err := json.Marshal(anthropicRequest{
		Model:     p.Model,
		System:    req.System,
		Messages:  []anthropicMessage{{Role: "user", Content: req.Prompt}},
		MaxTokens: maxTokens,
		Stream:    streaming,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	if streaming {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	resp, err := p.client().Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read body

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API returned status: %s", resp.Status)
	}

	if streaming {
		return p.consumeSSE(ctx, resp, req.OnToken)
	}

	var anthroResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthroResp); err != nil {
		return nil, err
	}
	if len(anthroResp.Content) == 0 {
		return nil, fmt.Errorf("anthropic API returned no content")
	}
	return &ai.CompletionResponse{
		Text:       anthroResp.Content[0].Text,
		Model:      p.Model,
		Usage:      ai.TokenUsage{InputTokens: anthroResp.Usage.InputTokens, OutputTokens: anthroResp.Usage.OutputTokens},
		Confidence: confidenceFromStopReason(anthroResp.StopReason),
	}, nil
}

// consumeSSE parses the Anthropic streaming response, invokes onToken per
// content_block_delta, and returns the assembled text + final usage. Honours
// ctx.Done so cancellation propagates promptly.
func (p *AnthropicProvider) consumeSSE(ctx context.Context, resp *http.Response, onToken func(string)) (*ai.CompletionResponse, error) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var assembled strings.Builder
	var usage ai.TokenUsage
	var stopReason string

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

		var ev anthropicSSEDelta
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue // skip malformed lines rather than abort the stream
		}

		switch ev.Type {
		case "message_start":
			usage.InputTokens = ev.Message.Usage.InputTokens
			usage.OutputTokens = ev.Message.Usage.OutputTokens
		case "content_block_delta":
			if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
				assembled.WriteString(ev.Delta.Text)
				if onToken != nil {
					onToken(ev.Delta.Text)
				}
			}
		case "message_delta":
			if ev.Delta.StopReason != "" {
				stopReason = ev.Delta.StopReason
			}
			if ev.Usage.OutputTokens > 0 {
				usage.OutputTokens = ev.Usage.OutputTokens
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("anthropic stream read: %w", err)
	}
	if assembled.Len() == 0 {
		return nil, fmt.Errorf("anthropic stream returned no content")
	}

	return &ai.CompletionResponse{
		Text:       assembled.String(),
		Model:      p.Model,
		Usage:      usage,
		Confidence: confidenceFromStopReason(stopReason),
	}, nil
}

// confidenceFromStopReason maps provider stop signals to a coarse 0..1
// confidence score. end_turn / stop = high confidence; max_tokens or
// length = the model was cut off mid-thought.
func confidenceFromStopReason(reason string) float32 {
	switch strings.ToLower(reason) {
	case "end_turn", "stop", "stop_sequence":
		return 1.0
	case "max_tokens", "length":
		return 0.5
	case "tool_use":
		return 0.9
	default:
		return 0
	}
}
