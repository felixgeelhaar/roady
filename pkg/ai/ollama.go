package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/felixgeelhaar/roady/pkg/domain/ai"
)

type OllamaProvider struct {
	Model      string
	baseURL    string       // overridable for tests; "" => http://localhost:11434
	httpClient *http.Client // overridable for tests; nil => http.DefaultClient
}

func NewOllamaProvider(model string) *OllamaProvider {
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{Model: model}
}

// NewOllamaProviderWithClient lets tests inject an httptest server.
func NewOllamaProviderWithClient(model, baseURL string, client *http.Client) *OllamaProvider {
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{Model: model, baseURL: baseURL, httpClient: client}
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

	url := p.baseURL
	if url == "" {
		url = "http://localhost:11434"
	}
	url = strings.TrimRight(url, "/") + "/api/generate"

	format := ""
	if strings.Contains(req.Prompt, "JSON") || strings.Contains(req.System, "JSON") {
		format = "json"
	}

	streaming := req.IsStreaming()

	body, err := json.Marshal(ollamaRequest{
		Model:  p.Model,
		Prompt: req.Prompt,
		System: req.System,
		Stream: streaming,
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

	client := p.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(hReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama API: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read body

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API error: status %d", resp.StatusCode)
	}

	if streaming {
		return p.consumeNDJSON(ctx, resp, req.OnToken)
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

// consumeNDJSON reads Ollama's newline-delimited JSON stream. Each line is
// an ollamaResponse; the final one has Done=true.
func (p *OllamaProvider) consumeNDJSON(ctx context.Context, resp *http.Response, onToken func(string)) (*ai.CompletionResponse, error) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var assembled strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk ollamaResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Response != "" {
			assembled.WriteString(chunk.Response)
			if onToken != nil {
				onToken(chunk.Response)
			}
		}
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ollama stream read: %w", err)
	}
	if assembled.Len() == 0 {
		return nil, fmt.Errorf("ollama stream returned no content")
	}

	text := strings.TrimSpace(assembled.String())
	return &ai.CompletionResponse{
		Text:  text,
		Model: p.Model,
		Usage: ai.TokenUsage{
			InputTokens:  0, // Ollama doesn't return token counts on the stream.
			OutputTokens: len(text) / 4,
		},
	}, nil
}
