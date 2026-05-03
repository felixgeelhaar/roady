package ai

import (
	"context"
	"strings"
	"testing"

	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
)

func TestMockProvider_StreamingChunksReassembleToFullText(t *testing.T) {
	p := &MockProvider{Model: "stream-test"}

	var chunks []string
	req := domainai.CompletionRequest{
		Prompt: "say hello",
		OnToken: func(chunk string) {
			chunks = append(chunks, chunk)
		},
	}

	if !req.IsStreaming() {
		t.Fatal("IsStreaming() = false, want true when OnToken set")
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("expected multi-chunk stream for prose, got %d chunks: %v", len(chunks), chunks)
	}
	if joined := strings.Join(chunks, ""); joined != resp.Text {
		t.Errorf("reassembled chunks = %q, want %q", joined, resp.Text)
	}
}

func TestMockProvider_JSONResponseStreamsAsSingleChunk(t *testing.T) {
	p := &MockProvider{Model: "stream-test"}

	var chunks []string
	req := domainai.CompletionRequest{
		// The mock special-cases prompts that mention JSON.
		Prompt:  "Return JSON tasks",
		OnToken: func(chunk string) { chunks = append(chunks, chunk) },
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected JSON to stream as a single chunk, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != resp.Text {
		t.Errorf("chunk = %q, response text = %q", chunks[0], resp.Text)
	}
}

func TestMockProvider_NonStreamingCallerUnaffected(t *testing.T) {
	p := &MockProvider{Model: "stream-test"}

	// No OnToken supplied — must behave exactly as before.
	req := domainai.CompletionRequest{Prompt: "anything"}
	if req.IsStreaming() {
		t.Fatal("IsStreaming() = true, want false when OnToken nil")
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp == nil || resp.Text == "" {
		t.Fatal("expected non-empty response in non-streaming mode")
	}
}

func TestMockProvider_StreamingHonoursCancelledContext(t *testing.T) {
	p := &MockProvider{Model: "stream-test"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	chunks := 0
	req := domainai.CompletionRequest{
		Prompt:  "this is a long prose response that has many words",
		OnToken: func(chunk string) { chunks++ },
	}

	if _, err := p.Complete(ctx, req); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Mock checks ctx.Done() before each emission. With a pre-cancelled
	// context, no chunks should be delivered.
	if chunks != 0 {
		t.Errorf("expected 0 chunks under cancelled context, got %d", chunks)
	}
}
