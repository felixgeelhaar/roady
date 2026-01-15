package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infraAI "github.com/felixgeelhaar/roady/pkg/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/ai"
)

func TestOpenAIProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hello from OpenAI!"}},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
			},
		})
	}))
	defer server.Close()

	p := infraAI.NewOpenAIProviderWithClient("gpt-4", "test-key", server.URL, server.Client())
	resp, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Text != "Hello from OpenAI!" {
		t.Errorf("expected 'Hello from OpenAI!', got %q", resp.Text)
	}
	if resp.Model != "gpt-4" {
		t.Errorf("expected model gpt-4, got %s", resp.Model)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("expected 5 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

func TestOpenAIProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "OK"}},
			},
			"usage": map[string]int{"prompt_tokens": 5, "completion_tokens": 1},
		})
	}))
	defer server.Close()

	p := infraAI.NewOpenAIProviderWithClient("gpt-4", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{
		Prompt: "Hello",
		System: "You are a helpful assistant",
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	messages := receivedBody["messages"].([]interface{})
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(messages))
	}

	systemMsg := messages[0].(map[string]interface{})
	if systemMsg["role"] != "system" {
		t.Errorf("expected first message role 'system', got %v", systemMsg["role"])
	}
	if systemMsg["content"] != "You are a helpful assistant" {
		t.Errorf("unexpected system content: %v", systemMsg["content"])
	}
}

func TestOpenAIProvider_Complete_NoAPIKey(t *testing.T) {
	p := infraAI.NewOpenAIProvider("gpt-4", "")
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestOpenAIProvider_Complete_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := infraAI.NewOpenAIProviderWithClient("gpt-4", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestOpenAIProvider_Complete_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := infraAI.NewOpenAIProviderWithClient("gpt-4", "invalid-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
}

func TestOpenAIProvider_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{},
			"usage":   map[string]int{"prompt_tokens": 5, "completion_tokens": 0},
		})
	}))
	defer server.Close()

	p := infraAI.NewOpenAIProviderWithClient("gpt-4", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestOpenAIProvider_Complete_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	p := infraAI.NewOpenAIProviderWithClient("gpt-4", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestOpenAIProvider_Complete_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never responds - simulates slow server
		select {}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	p := infraAI.NewOpenAIProviderWithClient("gpt-4", "test-key", server.URL, server.Client())
	_, err := p.Complete(ctx, ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestOpenAIProvider_DefaultModel(t *testing.T) {
	p := infraAI.NewOpenAIProvider("", "test-key")
	if p.ID() != "openai:gpt-4o" {
		t.Errorf("expected default model gpt-4o, got %s", p.ID())
	}
}

func TestOpenAIProviderWithClient_DefaultModel(t *testing.T) {
	p := infraAI.NewOpenAIProviderWithClient("", "test-key", "", nil)
	if p.ID() != "openai:gpt-4o" {
		t.Errorf("expected default model gpt-4o, got %s", p.ID())
	}
}
