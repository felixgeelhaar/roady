package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (p *GeminiProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key not provided (set GEMINI_API_KEY)")
	}

	gReq := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: req.Prompt}}},
		},
	}

	if req.System != "" {
		gReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, err
	}

	url := p.baseURL
	if url == "" {
		url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.Model, p.APIKey)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := p.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned status: %s", resp.Status)
	}

	var gResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return nil, err
	}

	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini API returned no candidates")
	}

	return &ai.CompletionResponse{
		Text:  gResp.Candidates[0].Content.Parts[0].Text,
		Model: p.Model,
		Usage: ai.TokenUsage{
			InputTokens:  gResp.UsageMetadata.PromptTokenCount,
			OutputTokens: gResp.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}
