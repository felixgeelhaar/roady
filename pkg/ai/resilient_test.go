package ai_test

import (
	"testing"
	"time"

	infraAI "github.com/felixgeelhaar/roady/pkg/ai"
)

func TestResilientProvider_ID_Delegates(t *testing.T) {
	inner := &infraAI.MockProvider{Model: "test-model"}
	p := infraAI.NewResilientProvider(inner)
	if p.ID() != "mock:test-model" {
		t.Errorf("expected ID 'mock:test-model', got %q", p.ID())
	}
}

func TestResilientProvider_DefaultConfig(t *testing.T) {
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

func TestResilientProvider_ZeroConfig(t *testing.T) {
	inner := &infraAI.MockProvider{Model: "test"}
	// Zero config should get defaults applied
	p := infraAI.NewResilientProviderWithConfig(inner, infraAI.ResilienceConfig{})
	if p.ID() != "mock:test" {
		t.Errorf("expected ID 'mock:test', got %q", p.ID())
	}
}

func TestResilientProvider_CustomConfig(t *testing.T) {
	inner := &infraAI.MockProvider{Model: "test"}
	cfg := infraAI.ResilienceConfig{
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
		Timeout:    60 * time.Second,
	}
	p := infraAI.NewResilientProviderWithConfig(inner, cfg)
	if p.ID() != "mock:test" {
		t.Errorf("expected ID 'mock:test', got %q", p.ID())
	}
}
