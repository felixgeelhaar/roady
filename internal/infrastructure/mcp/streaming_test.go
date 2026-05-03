package mcp

import (
	"context"
	"sync"
	"testing"

	mcplib "github.com/felixgeelhaar/mcp-go"
	mcpserver "github.com/felixgeelhaar/mcp-go/server"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
)

// fakeProgressReporter records every ReportWithMessage call so tests can
// assert that streaming chunks reached the MCP progress channel.
type fakeProgressReporter struct {
	token mcplib.ProgressToken

	mu       sync.Mutex
	messages []string
}

func (f *fakeProgressReporter) Report(_ float64, _ *float64) error { return nil }

func (f *fakeProgressReporter) ReportWithMessage(_ float64, _ *float64, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, message)
	return nil
}

func (f *fakeProgressReporter) Token() mcplib.ProgressToken { return f.token }

func (f *fakeProgressReporter) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.messages))
	copy(out, f.messages)
	return out
}

func TestWithMCPStreaming_NoTokenIsNoOp(t *testing.T) {
	ctx := context.Background()
	out := withMCPStreaming(ctx)
	if domainai.OnTokenFromContext(out) != nil {
		t.Error("expected no OnToken sink when client did not subscribe")
	}
}

func TestWithMCPStreaming_ForwardsChunksToProgressReporter(t *testing.T) {
	rep := &fakeProgressReporter{token: mcplib.ProgressToken("tok-123")}

	ctx := mcpserver.ContextWithProgress(context.Background(), rep)
	ctx = withMCPStreaming(ctx)

	cb := domainai.OnTokenFromContext(ctx)
	if cb == nil {
		t.Fatal("expected OnToken sink to be installed when token present")
	}

	cb("Hello")
	cb(" world")

	got := rep.snapshot()
	if len(got) != 2 || got[0] != "Hello" || got[1] != " world" {
		t.Fatalf("messages = %v, want [Hello, ' world']", got)
	}
}
