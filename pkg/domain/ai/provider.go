package ai

import (
	"context"
)

// CompletionRequest represents a prompt to the AI.
type CompletionRequest struct {
	Prompt      string
	System      string
	Temperature float32
	MaxTokens   int

	// OnToken is an optional callback invoked by streaming-capable
	// providers as tokens arrive. The chunk is the new text since the
	// last call (not the cumulative text). Providers that do not stream
	// MUST ignore this field; the final assembled Text is still returned
	// in CompletionResponse so callers do not need to assemble chunks
	// themselves. Callers that do not need streaming leave it nil.
	OnToken func(chunk string) `json:"-"`
}

// IsStreaming reports whether the request opts in to streaming. Helper
// avoids nil-checks at every provider call site.
func (r CompletionRequest) IsStreaming() bool { return r.OnToken != nil }

// CompletionResponse represents the AI's answer.
type CompletionResponse struct {
	Text       string
	Usage      TokenUsage
	Model      string
	Confidence float32     // optional 0..1 self-rated confidence; 0 means unknown
	Sources    []SourceRef // optional supporting citations
}

// TokenUsage tracks costs.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// SourceRef points back to the document evidence the model relied on. Fields
// are intentionally minimal: a stable doc identifier plus a human-readable
// span. Providers that cannot produce citations leave the slice nil.
type SourceRef struct {
	Doc  string `json:"doc"`            // file path, URL, or stable id
	Span string `json:"span,omitempty"` // e.g. "lines 12-18" or heading anchor
	Note string `json:"note,omitempty"` // free-form annotation
}

// HasConfidence reports whether the response carries a usable confidence
// score. Providers signal "unknown" by leaving Confidence at 0.
func (r *CompletionResponse) HasConfidence() bool {
	return r != nil && r.Confidence > 0
}

// Provider is the interface for all AI backends.
type Provider interface {
	ID() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// ctxOnTokenKey is a private context-value key used by WithOnToken /
// OnTokenFromContext so callers can opt into streaming once at a CLI or
// MCP layer and have it ride through every nested provider call without
// changing service signatures.
type ctxOnTokenKey struct{}

// WithOnToken returns a child context carrying the supplied streaming
// callback. Service-layer code reads it via OnTokenFromContext and applies
// it to outbound CompletionRequests.
func WithOnToken(ctx context.Context, fn func(chunk string)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxOnTokenKey{}, fn)
}

// OnTokenFromContext returns the streaming callback installed by
// WithOnToken, or nil when none is present.
func OnTokenFromContext(ctx context.Context) func(chunk string) {
	if v := ctx.Value(ctxOnTokenKey{}); v != nil {
		if fn, ok := v.(func(string)); ok {
			return fn
		}
	}
	return nil
}
