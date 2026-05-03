package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
)

func TestAnthropicProvider_Streaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing API key header: %v", r.Header)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":12,"output_tokens":1}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}

event: message_stop
data: {"type":"message_stop"}
`))
	}))
	defer srv.Close()

	p := NewAnthropicProviderWithClient("claude-test", "test-key", srv.URL, srv.Client())

	var chunks []string
	resp, err := p.Complete(context.Background(), domainai.CompletionRequest{
		Prompt:  "hi",
		OnToken: func(c string) { chunks = append(chunks, c) },
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "Hello world" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello world")
	}
	if len(chunks) != 2 || chunks[0] != "Hello" || chunks[1] != " world" {
		t.Errorf("chunks = %v, want [Hello, ' world']", chunks)
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 7 {
		t.Errorf("usage = %+v", resp.Usage)
	}
	if resp.Confidence != 1.0 {
		t.Errorf("confidence = %v, want 1.0 for end_turn", resp.Confidence)
	}
}

func TestOpenAIProvider_Streaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"Hi"}}]}

data: {"choices":[{"delta":{"content":" there"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2}}

data: [DONE]
`))
	}))
	defer srv.Close()

	p := NewOpenAIProviderWithClient("gpt-test", "test-key", srv.URL, srv.Client())

	var chunks []string
	resp, err := p.Complete(context.Background(), domainai.CompletionRequest{
		Prompt:  "hi",
		OnToken: func(c string) { chunks = append(chunks, c) },
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "Hi there" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hi there")
	}
	if strings.Join(chunks, "") != "Hi there" {
		t.Errorf("chunks = %v", chunks)
	}
	if resp.Confidence != 1.0 {
		t.Errorf("confidence = %v, want 1.0 for stop", resp.Confidence)
	}
}

func TestOllamaProvider_Streaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":"Hello","done":false}
{"response":" world","done":false}
{"response":"","done":true}
`))
	}))
	defer srv.Close()

	p := NewOllamaProviderWithClient("llama3", srv.URL, srv.Client())

	var chunks []string
	resp, err := p.Complete(context.Background(), domainai.CompletionRequest{
		Prompt:  "hi",
		OnToken: func(c string) { chunks = append(chunks, c) },
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "Hello world" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello world")
	}
	if strings.Join(chunks, "") != "Hello world" {
		t.Errorf("chunks = %v", chunks)
	}
}

func TestGeminiProvider_Streaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}

data: {"candidates":[{"content":{"parts":[{"text":" world"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3}}
`))
	}))
	defer srv.Close()

	p := NewGeminiProviderWithClient("gemini-test", "test-key", srv.URL, srv.Client())

	var chunks []string
	resp, err := p.Complete(context.Background(), domainai.CompletionRequest{
		Prompt:  "hi",
		OnToken: func(c string) { chunks = append(chunks, c) },
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "Hello world" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello world")
	}
	if strings.Join(chunks, "") != "Hello world" {
		t.Errorf("chunks = %v", chunks)
	}
	if resp.Confidence != 1.0 {
		t.Errorf("confidence = %v, want 1.0 for STOP", resp.Confidence)
	}
}

func TestConfidenceFromStopReason_Mapping(t *testing.T) {
	cases := map[string]float32{
		"end_turn":      1.0,
		"stop":          1.0,
		"stop_sequence": 1.0,
		"max_tokens":    0.5,
		"length":        0.5,
		"tool_use":      0.9,
		"":              0,
		"unknown":       0,
	}
	for reason, want := range cases {
		if got := confidenceFromStopReason(reason); got != want {
			t.Errorf("confidenceFromStopReason(%q) = %v, want %v", reason, got, want)
		}
	}
}
