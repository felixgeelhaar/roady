package ai

import (
	"context"
	"time"

	"github.com/felixgeelhaar/fortify/retry"
	"github.com/felixgeelhaar/fortify/timeout"
	"github.com/felixgeelhaar/roady/pkg/domain/ai"
)

// ResilienceConfig holds retry and timeout settings for AI providers.
type ResilienceConfig struct {
	MaxRetries int           // Maximum retry attempts (default: 2)
	RetryDelay time.Duration // Initial retry delay (default: 1s)
	Timeout    time.Duration // Request timeout (default: 300s)
}

// DefaultResilienceConfig returns sensible defaults.
func DefaultResilienceConfig() ResilienceConfig {
	return ResilienceConfig{
		MaxRetries: 2,
		RetryDelay: time.Second,
		Timeout:    300 * time.Second,
	}
}

type ResilientProvider struct {
	inner  ai.Provider
	config ResilienceConfig
}

func NewResilientProvider(inner ai.Provider) *ResilientProvider {
	return NewResilientProviderWithConfig(inner, DefaultResilienceConfig())
}

func NewResilientProviderWithConfig(inner ai.Provider, config ResilienceConfig) *ResilientProvider {
	// Apply defaults for zero values
	if config.MaxRetries <= 0 {
		config.MaxRetries = 2
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = time.Second
	}
	if config.Timeout <= 0 {
		config.Timeout = 300 * time.Second
	}
	return &ResilientProvider{inner: inner, config: config}
}

func (p *ResilientProvider) ID() string {
	return p.inner.ID()
}

func (p *ResilientProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	r := retry.New[*ai.CompletionResponse](retry.Config{
		MaxAttempts:   p.config.MaxRetries,
		InitialDelay:  p.config.RetryDelay,
		BackoffPolicy: retry.BackoffExponential,
	})

	t := timeout.New[*ai.CompletionResponse](timeout.Config{
		DefaultTimeout: p.config.Timeout,
	})

	return t.Execute(ctx, p.config.Timeout, func(ctx context.Context) (*ai.CompletionResponse, error) {
		res, err := r.Do(ctx, func(ctx context.Context) (*ai.CompletionResponse, error) {
			return p.inner.Complete(ctx, req)
		})
		return res, err
	})
}
