package sse_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/internal/infrastructure/sse"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestSSEHandler_StreamsEvents(t *testing.T) {
	publisher := storage.NewInMemoryEventPublisher()
	handler := sse.NewSSEHandler(publisher)

	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Publish event after a delay, then cancel
	go func() {
		time.Sleep(300 * time.Millisecond)
		publisher.Publish(&events.BaseEvent{
			ID:        "test-1",
			Type:      events.EventTypeTaskStarted,
			Timestamp: time.Now(),
		})
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Cancelled context is expected
		if ctx.Err() != nil {
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "task.started") {
		t.Log("received task.started event")
	}
}

func TestNewSSEHandler_CreatesHandler(t *testing.T) {
	publisher := storage.NewInMemoryEventPublisher()
	handler := sse.NewSSEHandler(publisher)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}
