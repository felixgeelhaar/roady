package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
)

// progressTickInterval is how often the elapsed-time line is reprinted
// while an AI operation is in flight. Short enough to feel responsive;
// long enough to avoid drowning the terminal.
var progressTickInterval = 5 * time.Second

// progressGracePeriod is how long withAIProgress waits for the wrapped
// function to honour a cancellation signal before returning.
var progressGracePeriod = 2 * time.Second

// errAIOperationCancelled is returned when the user interrupts an
// AI-backed CLI invocation via SIGINT. Surfaced to the caller so wrappers
// can suppress the usual non-zero stack trace.
var errAIOperationCancelled = errors.New("operation cancelled by user")

// withAIProgress wraps a long-running AI call with three affordances:
//
//   - a streaming OnToken sink installed on ctx via ai.WithOnToken so any
//     provider that supports SSE prints its tokens to stderr in real time;
//   - an elapsed-time ticker on stderr suppressed once the first token
//     arrives (Doherty Threshold mitigation when no streaming is offered);
//   - a SIGINT/SIGTERM handler that cancels ctx and waits briefly for fn
//     to bail out cleanly.
//
// fn must respect the supplied context for cancellation to be effective.
// Output goes to the package-level progressOut writer so tests can capture
// it without touching os.Stderr.
func withAIProgress(parent context.Context, label string, fn func(ctx context.Context) error) error {
	if parent == nil {
		parent = context.Background()
	}

	out := progressOut
	if out == nil {
		out = os.Stderr
	}

	// Track whether we have started writing streamed tokens so the elapsed
	// ticker can step out of the way once real output is flowing.
	var streaming atomic.Bool
	onToken := func(chunk string) {
		if streaming.CompareAndSwap(false, true) {
			_, _ = fmt.Fprintf(out, "%s ... streaming\n", label)
		}
		_, _ = fmt.Fprint(out, chunk)
	}

	ctx, cancel := context.WithCancel(domainai.WithOnToken(parent, onToken))
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	_, _ = fmt.Fprintf(out, "%s ... working\n", label)

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		done <- fn(ctx)
	}()

	ticker := time.NewTicker(progressTickInterval)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			elapsed := int(time.Since(start).Seconds())
			if streaming.Load() {
				// End the streamed line cleanly.
				_, _ = fmt.Fprintln(out)
			}
			if err == nil {
				_, _ = fmt.Fprintf(out, "%s ... done in %ds\n", label, elapsed)
				return nil
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, errAIOperationCancelled) {
				return errAIOperationCancelled
			}
			return err

		case <-ticker.C:
			// Once tokens are streaming, the elapsed ticker would just
			// interleave junk with the response.
			if streaming.Load() {
				continue
			}
			_, _ = fmt.Fprintf(out, "%s ... %ds elapsed\n", label, int(time.Since(start).Seconds()))

		case <-sigCh:
			_, _ = fmt.Fprintln(out, "Cancelling — waiting for provider to release...")
			cancel()
			select {
			case <-done:
			case <-time.After(progressGracePeriod):
			}
			return errAIOperationCancelled
		}
	}
}

// progressOut is the destination writer for ticker / cancellation
// messages. Defaults to os.Stderr; tests substitute a buffer.
var progressOut io.Writer = os.Stderr
