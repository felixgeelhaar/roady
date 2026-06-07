package mcp

import (
	"context"

	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	mcplib "go.klarlabs.de/mcp"
)

// withMCPStreaming returns a context that, when the MCP request carries a
// progress token, installs an OnToken sink (via ai.WithOnToken) that
// forwards each streamed chunk as an MCP progress notification. Callers
// without a progress token get the parent context unchanged so behaviour
// is identical for non-subscribing clients.
//
// Progress is reported as a monotonically increasing chunk count because
// token streams have no natural total to report against. The chunk text
// rides along as the progress message so interactive clients can display
// it directly.
func withMCPStreaming(ctx context.Context) context.Context {
	pr := mcplib.ProgressFromContext(ctx)
	// ProgressFromContext returns a no-op reporter (Token() == "") when
	// the client did not subscribe; treat that as "no progress wanted".
	if pr == nil || pr.Token() == "" {
		return ctx
	}

	var count float64
	cb := func(chunk string) {
		count++
		// Report errors are non-fatal: a notification dropped in flight
		// must never abort the AI call. The progress reporter itself is
		// thread-safe per its docs.
		_ = pr.ReportWithMessage(count, nil, chunk)
	}
	return domainai.WithOnToken(ctx, cb)
}
