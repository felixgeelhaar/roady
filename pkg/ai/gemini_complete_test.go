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

func TestGeminiProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": "Hello from Gemini!"},
						},
					},
				},
			},
			"usageMetadata": map[string]int{
				"promptTokenCount":     10,
				"candidatesTokenCount": 5,
			},
		})
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	resp, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Text != "Hello from Gemini!" {
		t.Errorf("expected 'Hello from Gemini!', got %q", resp.Text)
	}
	if resp.Model != "gemini-1.5-pro" {
		t.Errorf("expected model gemini-1.5-pro, got %s", resp.Model)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("expected 5 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

func TestGeminiProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{{"text": "OK"}},
					},
				},
			},
			"usageMetadata": map[string]int{"promptTokenCount": 5, "candidatesTokenCount": 1},
		})
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{
		Prompt: "Hello",
		System: "You are a helpful assistant",
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Verify system instruction was included
	if receivedBody["system_instruction"] == nil {
		t.Fatal("expected system_instruction in request")
	}

	systemInstruction := receivedBody["system_instruction"].(map[string]interface{})
	parts := systemInstruction["parts"].([]interface{})
	if len(parts) == 0 {
		t.Fatal("expected parts in system_instruction")
	}
	part := parts[0].(map[string]interface{})
	if part["text"] != "You are a helpful assistant" {
		t.Errorf("unexpected system text: %v", part["text"])
	}
}

func TestGeminiProvider_Complete_NoAPIKey(t *testing.T) {
	p := infraAI.NewGeminiProvider("gemini-1.5-pro", "")
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestGeminiProvider_Complete_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestGeminiProvider_Complete_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "invalid-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
}

func TestGeminiProvider_Complete_EmptyCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates":    []map[string]interface{}{},
			"usageMetadata": map[string]int{"promptTokenCount": 5, "candidatesTokenCount": 0},
		})
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
}

func TestGeminiProvider_Complete_EmptyParts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{},
					},
				},
			},
			"usageMetadata": map[string]int{"promptTokenCount": 5, "candidatesTokenCount": 0},
		})
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for empty parts")
	}
}

func TestGeminiProvider_Complete_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestGeminiProvider_Complete_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never responds - simulates slow server
		select {}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	_, err := p.Complete(ctx, ai.CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGeminiProvider_DefaultModel(t *testing.T) {
	p := infraAI.NewGeminiProvider("", "test-key")
	if p.ID() != "gemini:gemini-1.5-pro" {
		t.Errorf("expected default model gemini-1.5-pro, got %s", p.ID())
	}
}

func TestGeminiProviderWithClient_DefaultModel(t *testing.T) {
	p := infraAI.NewGeminiProviderWithClient("", "test-key", "", nil)
	if p.ID() != "gemini:gemini-1.5-pro" {
		t.Errorf("expected default model gemini-1.5-pro, got %s", p.ID())
	}
}

func TestGeminiProvider_Complete_NoSystemPrompt(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{{"text": "OK"}},
					},
				},
			},
			"usageMetadata": map[string]int{"promptTokenCount": 5, "candidatesTokenCount": 1},
		})
	}))
	defer server.Close()

	p := infraAI.NewGeminiProviderWithClient("gemini-1.5-pro", "test-key", server.URL, server.Client())
	_, err := p.Complete(context.Background(), ai.CompletionRequest{
		Prompt: "Hello",
		// No System prompt
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Verify system instruction was NOT included
	if receivedBody["system_instruction"] != nil {
		t.Error("expected no system_instruction in request when System is empty")
	}
}
