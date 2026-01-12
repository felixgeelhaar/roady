package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/felixgeelhaar/roady/internal/domain/ai"
)

type OllamaProvider struct {
	Model string
}

func NewOllamaProvider(model string) *OllamaProvider {
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{Model: model}
}

func (p *OllamaProvider) ID() string {
	return "ollama:" + p.Model
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

var safeModelName = regexp.MustCompile(`^[a-zA-Z0-9:._-]+$`)

func (p *OllamaProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if !safeModelName.MatchString(p.Model) {
		return nil, fmt.Errorf("invalid model name: %s", p.Model)
	}

	if req.Temperature < 0 {
		return nil, fmt.Errorf("invalid temperature")
	}

	url := "http://localhost:11434/api/generate"
	
	format := ""
	if strings.Contains(req.Prompt, "JSON") || strings.Contains(req.System, "JSON") {
		format = "json"
	}

	body, err := json.Marshal(ollamaRequest{
		Model:  p.Model,
		Prompt: req.Prompt,
		System: req.System,
		Stream: false,
		Format: format,
	})
	if err != nil {
		return nil, err
	}

	hReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	hReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(hReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API error: status %d", resp.StatusCode)
	}

	var oResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&oResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return &ai.CompletionResponse{
		Text:  strings.TrimSpace(oResp.Response),
		Model: p.Model,
		Usage: ai.TokenUsage{
			InputTokens:  len(req.Prompt) / 4,
			OutputTokens: len(oResp.Response) / 4,
		},
	}, nil
}