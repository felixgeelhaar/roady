package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
)

func withProgressOut(t *testing.T) *bytes.Buffer {
	t.Helper()
	prev := progressOut
	buf := new(bytes.Buffer)
	progressOut = buf
	t.Cleanup(func() { progressOut = prev })
	return buf
}

func TestWithAIProgress_Success(t *testing.T) {
	buf := withProgressOut(t)

	err := withAIProgress(context.Background(), "Test op", func(_ context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Test op ... working") {
		t.Errorf("expected start banner, got: %q", out)
	}
	if !strings.Contains(out, "Test op ... done in") {
		t.Errorf("expected completion line, got: %q", out)
	}
}

func TestWithAIProgress_PropagatesError(t *testing.T) {
	withProgressOut(t)

	wantErr := errors.New("boom")
	err := withAIProgress(context.Background(), "Failing op", func(_ context.Context) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestWithAIProgress_TickWhileRunning(t *testing.T) {
	prevInterval := progressTickInterval
	progressTickInterval = 20 * time.Millisecond
	t.Cleanup(func() { progressTickInterval = prevInterval })

	buf := withProgressOut(t)

	err := withAIProgress(context.Background(), "Slow op", func(_ context.Context) error {
		time.Sleep(60 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "elapsed") {
		t.Errorf("expected at least one tick line containing 'elapsed', got: %q", out)
	}
}

func TestWithAIProgress_StreamsTokensFromContext(t *testing.T) {
	buf := withProgressOut(t)

	err := withAIProgress(context.Background(), "Streaming op", func(ctx context.Context) error {
		// Simulate a streaming-capable provider that fetches the
		// callback the wrapper installed and emits chunks through it.
		cb := domainai.OnTokenFromContext(ctx)
		if cb == nil {
			t.Fatal("expected OnToken in context")
		}
		cb("Hello")
		cb(" world")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Hello world") {
		t.Errorf("expected streamed chunks in output, got: %q", out)
	}
	if !strings.Contains(out, "Streaming op ... streaming") {
		t.Errorf("expected streaming banner, got: %q", out)
	}
	if !strings.Contains(out, "Streaming op ... done in") {
		t.Errorf("expected completion line, got: %q", out)
	}
}

func TestWithAIProgress_HonoursCancelledContext(t *testing.T) {
	withProgressOut(t)

	// A pre-cancelled parent context should propagate cancellation to fn
	// and surface as the canonical cancellation sentinel.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := withAIProgress(ctx, "Cancelled op", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, errAIOperationCancelled) {
		t.Fatalf("error = %v, want %v", err, errAIOperationCancelled)
	}
}
