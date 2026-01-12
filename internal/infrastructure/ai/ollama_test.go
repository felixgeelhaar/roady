package ai_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/internal/domain/ai"
	infraAI "github.com/felixgeelhaar/roady/internal/infrastructure/ai"
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