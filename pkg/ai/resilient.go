package ai

import (
	"context"
	"time"

	"github.com/felixgeelhaar/fortify/retry"
	"github.com/felixgeelhaar/fortify/timeout"
	"github.com/felixgeelhaar/roady/pkg/domain/ai"
)

type ResilientProvider struct {
	inner ai.Provider
}

func NewResilientProvider(inner ai.Provider) *ResilientProvider {
	return &ResilientProvider{inner: inner}
}

func (p *ResilientProvider) ID() string {
	return p.inner.ID()
}

func (p *ResilientProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	// Configure Retry
	r := retry.New[*ai.CompletionResponse](retry.Config{
		MaxAttempts:   2,
		InitialDelay:  time.Second,
		BackoffPolicy: retry.BackoffExponential,
	})

	// Configure Timeout
	t := timeout.New[*ai.CompletionResponse](timeout.Config{
		DefaultTimeout: 300 * time.Second,
	})

	// Execute with both
	// Timeout.Execute takes (ctx, duration, func). If duration is 0, it might use default or no extra timeout.
	// Since we set DefaultTimeout in config, passing 0 here might work if the library supports it, 
	// otherwise we pass the config value again or rely on the Config's behavior.
	// Based on typical patterns, let's pass the same 300s or 0.
	return t.Execute(ctx, 300*time.Second, func(ctx context.Context) (*ai.CompletionResponse, error) {
		res, err := r.Do(ctx, func(ctx context.Context) (*ai.CompletionResponse, error) {
			return p.inner.Complete(ctx, req)
		})
		return res, err
	})
}
