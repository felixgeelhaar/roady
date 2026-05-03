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

type GeminiProvider struct {
	Model      string
	APIKey     string
	baseURL    string       // For testing - if set, used directly; otherwise uses default Gemini URL
	httpClient *http.Client // For testing - defaults to http.DefaultClient
}

func NewGeminiProvider(model string, apiKey string) *GeminiProvider {
	if model == "" {
		model = "gemini-1.5-pro"
	}
	return &GeminiProvider{
		Model:  model,
		APIKey: apiKey,
	}
}

// NewGeminiProviderWithClient creates a provider with custom HTTP client and base URL (for testing).
func NewGeminiProviderWithClient(model, apiKey, baseURL string, client *http.Client) *GeminiProvider {
	if model == "" {
		model = "gemini-1.5-pro"
	}
	return &GeminiProvider{
		Model:      model,
		APIKey:     apiKey,
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (p *GeminiProvider) ID() string {
	return "gemini:" + p.Model
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content       geminiContent `json:"content"`
		FinishReason  string        `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

// generateURL returns the non-streaming endpoint, honouring an injected
// baseURL for tests.
func (p *GeminiProvider) generateURL() string {
	if p.baseURL != "" {
		return p.baseURL
	}
	return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.Model, p.APIKey)
}

// streamURL returns the streaming endpoint. The Gemini API exposes
// streamGenerateContent which yields a JSON array; ?alt=sse switches it to
// proper SSE with `data:` lines.
func (p *GeminiProvider) streamURL() string {
	if p.baseURL != "" {
		// Tests inject a complete URL; reuse it as-is.
		return p.baseURL
	}
	return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", p.Model, p.APIKey)
}

func (p *GeminiProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("gemini API key not provided (set GEMINI_API_KEY)")
	}

	gReq := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: req.Prompt}}}},
	}
	if req.System != "" {
		gReq.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: req.System}}}
	}

	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, err
	}

	streaming := req.IsStreaming()

	url := p.generateURL()
	if streaming {
		url = p.streamURL()
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
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
		return nil, fmt.Errorf("gemini API returned status: %s", resp.Status)
	}

	if streaming {
		return p.consumeSSE(ctx, resp, req.OnToken)
	}

	var gResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return nil, err
	}
	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini API returned no candidates")
	}
	return &ai.CompletionResponse{
		Text:       gResp.Candidates[0].Content.Parts[0].Text,
		Model:      p.Model,
		Usage:      ai.TokenUsage{InputTokens: gResp.UsageMetadata.PromptTokenCount, OutputTokens: gResp.UsageMetadata.CandidatesTokenCount},
		Confidence: confidenceFromGeminiFinish(gResp.Candidates[0].FinishReason),
	}, nil
}

// consumeSSE parses Gemini's SSE-formatted streamGenerateContent response.
// Each `data: {...}` line is a partial geminiResponse with a text fragment.
func (p *GeminiProvider) consumeSSE(ctx context.Context, resp *http.Response, onToken func(string)) (*ai.CompletionResponse, error) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var assembled strings.Builder
	var usage ai.TokenUsage
	var finish string

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
		if payload == "" {
			continue
		}
		var chunk geminiResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, cand := range chunk.Candidates {
			for _, part := range cand.Content.Parts {
				if part.Text == "" {
					continue
				}
				assembled.WriteString(part.Text)
				if onToken != nil {
					onToken(part.Text)
				}
			}
			if cand.FinishReason != "" {
				finish = cand.FinishReason
			}
		}
		if chunk.UsageMetadata.PromptTokenCount > 0 {
			usage.InputTokens = chunk.UsageMetadata.PromptTokenCount
		}
		if chunk.UsageMetadata.CandidatesTokenCount > 0 {
			usage.OutputTokens = chunk.UsageMetadata.CandidatesTokenCount
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("gemini stream read: %w", err)
	}
	if assembled.Len() == 0 {
		return nil, fmt.Errorf("gemini stream returned no content")
	}

	return &ai.CompletionResponse{
		Text:       assembled.String(),
		Model:      p.Model,
		Usage:      usage,
		Confidence: confidenceFromGeminiFinish(finish),
	}, nil
}

// confidenceFromGeminiFinish maps Gemini's finishReason to a coarse 0..1
// confidence score. STOP is the natural-completion signal; MAX_TOKENS
// indicates truncation; SAFETY/RECITATION are content-policy stops.
func confidenceFromGeminiFinish(reason string) float32 {
	switch strings.ToUpper(reason) {
	case "STOP":
		return 1.0
	case "MAX_TOKENS":
		return 0.5
	case "SAFETY", "RECITATION", "OTHER":
		return 0.2
	default:
		return 0
	}
}
