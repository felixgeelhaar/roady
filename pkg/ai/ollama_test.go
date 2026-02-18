package ai_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	infraAI "github.com/felixgeelhaar/roady/pkg/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/ai"
)

func TestOllamaProvider_Basic(t *testing.T) {
	p := infraAI.NewOllamaProvider("")
	if p.ID() != "ollama:llama3" {
		t.Errorf("expected ID ollama:llama3, got %s", p.ID())
	}
}

func TestOllamaProvider_Validation(t *testing.T) {
	p := infraAI.NewOllamaProvider("invalid model;")
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "hi"})
	if err == nil {
		t.Error("expected error for invalid model name")
	}
}

func TestOllamaProvider_Temp(t *testing.T) {
	p := infraAI.NewOllamaProvider("llama3")
	_, err := p.Complete(context.Background(), ai.CompletionRequest{Temperature: -1})
	if err == nil {
		t.Error("expected error for negative temp")
	}
}

func TestOllamaProvider_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	p := infraAI.NewOllamaProvider("llama3")
	_, err := p.Complete(ctx, ai.CompletionRequest{Prompt: "hi"})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestOllamaProvider_SystemPrompt(t *testing.T) {
	p := infraAI.NewOllamaProvider("llama3")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, _ = p.Complete(ctx, ai.CompletionRequest{Prompt: "hi", System: "you are a bot"})
}

type StubProvider struct {
	Result string
	Err    error
}

func (s *StubProvider) ID() string { return "stub" }
func (s *StubProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	return &ai.CompletionResponse{Text: s.Result}, s.Err
}

func TestResilientProvider_Success(t *testing.T) {
	stub := &StubProvider{Result: "ok"}
	res := infraAI.NewResilientProvider(stub)

	resp, err := res.Complete(context.Background(), ai.CompletionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "ok" {
		t.Errorf("expected ok, got %s", resp.Text)
	}
}

type FaultyProvider struct {
	attempts int
	maxFail  int
}

func (f *FaultyProvider) ID() string { return "faulty" }
func (f *FaultyProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	f.attempts++
	if f.maxFail > 0 && f.attempts <= f.maxFail {
		return nil, errors.New("transient error")
	}
	if f.maxFail == -1 {
		return nil, errors.New("permanent error")
	}
	return &ai.CompletionResponse{Text: "success"}, nil
}

func TestResilientProvider_Retry(t *testing.T) {
	faulty := &FaultyProvider{maxFail: 1}
	resilient := infraAI.NewResilientProvider(faulty)

	resp, err := resilient.Complete(context.Background(), ai.CompletionRequest{})
	if err != nil {
		t.Fatalf("Expected success after retry, got: %v", err)
	}
	if resp.Text != "success" {
		t.Errorf("Expected success response")
	}
}

type SlowProvider struct{}

func (s *SlowProvider) ID() string { return "slow" }
func (s *SlowProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return &ai.CompletionResponse{Text: "too late"}, nil
	}
}

func TestResilientProvider_Timeout_Fail(t *testing.T) {
	slow := &SlowProvider{}
	resilient := infraAI.NewResilientProvider(slow)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := resilient.Complete(ctx, ai.CompletionRequest{})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestResilienceConfig_Defaults(t *testing.T) {
	cfg := infraAI.DefaultResilienceConfig()
	if cfg.MaxRetries != 2 {
		t.Errorf("expected MaxRetries 2, got %d", cfg.MaxRetries)
	}
	if cfg.RetryDelay != time.Second {
		t.Errorf("expected RetryDelay 1s, got %v", cfg.RetryDelay)
	}
	if cfg.Timeout != 300*time.Second {
		t.Errorf("expected Timeout 300s, got %v", cfg.Timeout)
	}
}

func TestResilientProviderWithConfig_CustomValues(t *testing.T) {
	stub := &StubProvider{Result: "ok"}
	cfg := infraAI.ResilienceConfig{
		MaxRetries: 5,
		RetryDelay: 500 * time.Millisecond,
		Timeout:    60 * time.Second,
	}
	res := infraAI.NewResilientProviderWithConfig(stub, cfg)

	resp, err := res.Complete(context.Background(), ai.CompletionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "ok" {
		t.Errorf("expected ok, got %s", resp.Text)
	}
}

func TestResilientProviderWithConfig_ZeroValuesUseDefaults(t *testing.T) {
	stub := &StubProvider{Result: "ok"}
	cfg := infraAI.ResilienceConfig{} // All zero values
	res := infraAI.NewResilientProviderWithConfig(stub, cfg)

	// Should work with defaults applied
	resp, err := res.Complete(context.Background(), ai.CompletionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "ok" {
		t.Errorf("expected ok, got %s", resp.Text)
	}
}

func TestResilientProviderWithConfig_CustomRetries(t *testing.T) {
	faulty := &FaultyProvider{maxFail: 3}
	cfg := infraAI.ResilienceConfig{
		MaxRetries: 5,
		RetryDelay: 10 * time.Millisecond, // Fast retries for test
		Timeout:    5 * time.Second,
	}
	resilient := infraAI.NewResilientProviderWithConfig(faulty, cfg)

	resp, err := resilient.Complete(context.Background(), ai.CompletionRequest{})
	if err != nil {
		t.Fatalf("Expected success with 5 retries, got: %v", err)
	}
	if resp.Text != "success" {
		t.Errorf("Expected success response")
	}
	if faulty.attempts != 4 { // 3 failures + 1 success
		t.Errorf("Expected 4 attempts, got %d", faulty.attempts)
	}
}

func TestResilientProvider_ID(t *testing.T) {
	stub := &StubProvider{Result: "ok"}
	res := infraAI.NewResilientProvider(stub)
	if res.ID() != "stub" {
		t.Errorf("expected ID stub, got %s", res.ID())
	}
}

// Factory tests
func TestNewProvider_Ollama(t *testing.T) {
	p, err := infraAI.NewProvider("ollama", "llama3")
	if err != nil {
		t.Fatalf("NewProvider(ollama) error: %v", err)
	}
	if p.ID() != "ollama:llama3" {
		t.Errorf("expected ID ollama:llama3, got %s", p.ID())
	}
}

func TestNewProvider_OllamaDefault(t *testing.T) {
	p, err := infraAI.NewProvider("", "")
	if err != nil {
		t.Fatalf("NewProvider('', '') error: %v", err)
	}
	if p.ID() != "ollama:llama3" {
		t.Errorf("expected default ID ollama:llama3, got %s", p.ID())
	}
}

func TestNewProvider_Mock(t *testing.T) {
	p, err := infraAI.NewProvider("mock", "test-model")
	if err != nil {
		t.Fatalf("NewProvider(mock) error: %v", err)
	}
	if p.ID() != "mock:test-model" {
		t.Errorf("expected ID mock:test-model, got %s", p.ID())
	}
}

func TestNewProvider_Unsupported(t *testing.T) {
	_, err := infraAI.NewProvider("unsupported-provider", "model")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestNewProvider_OpenAI(t *testing.T) {
	p, err := infraAI.NewProvider("openai", "gpt-4")
	if err != nil {
		t.Fatalf("NewProvider(openai) error: %v", err)
	}
	if p.ID() != "openai:gpt-4" {
		t.Errorf("expected ID openai:gpt-4, got %s", p.ID())
	}
}

func TestNewProvider_Anthropic(t *testing.T) {
	p, err := infraAI.NewProvider("anthropic", "claude-3")
	if err != nil {
		t.Fatalf("NewProvider(anthropic) error: %v", err)
	}
	if p.ID() != "anthropic:claude-3" {
		t.Errorf("expected ID anthropic:claude-3, got %s", p.ID())
	}
}

func TestNewProvider_Gemini(t *testing.T) {
	p, err := infraAI.NewProvider("gemini", "gemini-pro")
	if err != nil {
		t.Fatalf("NewProvider(gemini) error: %v", err)
	}
	if p.ID() != "gemini:gemini-pro" {
		t.Errorf("expected ID gemini:gemini-pro, got %s", p.ID())
	}
}

func TestGetDefaultProvider_EnvOverride(t *testing.T) {
	// Save current env
	oldProvider := os.Getenv("ROADY_AI_PROVIDER")
	oldModel := os.Getenv("ROADY_AI_MODEL")
	defer func() {
		if err := os.Setenv("ROADY_AI_PROVIDER", oldProvider); err != nil { t.Fatal(err) }
		if err := os.Setenv("ROADY_AI_MODEL", oldModel); err != nil { t.Fatal(err) }
	}()

	if err := os.Setenv("ROADY_AI_PROVIDER", "mock"); err != nil { t.Fatal(err) }
	if err := os.Setenv("ROADY_AI_MODEL", "env-model"); err != nil { t.Fatal(err) }

	p, err := infraAI.GetDefaultProvider("ollama", "llama3")
	if err != nil {
		t.Fatalf("GetDefaultProvider error: %v", err)
	}
	if p.ID() != "mock:env-model" {
		t.Errorf("expected ID mock:env-model (from env), got %s", p.ID())
	}
}

// MockProvider tests
func TestMockProvider_ID(t *testing.T) {
	p, _ := infraAI.NewProvider("mock", "test")
	if p.ID() != "mock:test" {
		t.Errorf("expected mock:test, got %s", p.ID())
	}
}

func TestMockProvider_Complete(t *testing.T) {
	p, _ := infraAI.NewProvider("mock", "test")
	resp, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("Complete error: %v", err)
	}
	if resp.Text == "" {
		t.Error("expected non-empty response")
	}
	if resp.Usage.InputTokens == 0 || resp.Usage.OutputTokens == 0 {
		t.Error("expected non-zero token usage")
	}
}

func TestMockProvider_CompleteJSON(t *testing.T) {
	p, _ := infraAI.NewProvider("mock", "test")
	resp, err := p.Complete(context.Background(), ai.CompletionRequest{Prompt: "Generate JSON"})
	if err != nil {
		t.Fatalf("Complete error: %v", err)
	}
	if resp.Text[0] != '[' {
		t.Errorf("expected JSON array response for JSON prompt, got: %s", resp.Text)
	}
}
