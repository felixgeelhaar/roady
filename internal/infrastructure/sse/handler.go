// Package sse provides Server-Sent Events streaming for roady events.
package sse

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// SSEHandler streams events via Server-Sent Events.
type SSEHandler struct {
	publisher *storage.InMemoryEventPublisher
	mu        sync.RWMutex
	clients   map[chan *events.BaseEvent]struct{}
}

// NewSSEHandler creates a new SSE handler subscribed to the publisher.
func NewSSEHandler(publisher *storage.InMemoryEventPublisher) *SSEHandler {
	h := &SSEHandler{
		publisher: publisher,
		clients:   make(map[chan *events.BaseEvent]struct{}),
	}

	publisher.Subscribe(func(e *events.BaseEvent) error {
		h.mu.RLock()
		defer h.mu.RUnlock()
		for ch := range h.clients {
			select {
			case ch <- e:
			default:
				// Drop if client is slow
			}
		}
		return nil
	})

	return h
}

// ServeHTTP handles SSE connections.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Parse type filters from query param
	typeFilter := make(map[string]bool)
	if types := r.URL.Query().Get("types"); types != "" {
		for _, t := range strings.Split(types, ",") {
			typeFilter[strings.TrimSpace(t)] = true
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan *events.BaseEvent, 64)

	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		close(ch)
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}

			// Apply type filter
			if len(typeFilter) > 0 && !typeFilter[event.Type] {
				continue
			}

			_, _ = fmt.Fprintf(w, "id: %s\n", event.ID)
			_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
			_, _ = fmt.Fprintf(w, "data: {\"type\":\"%s\",\"aggregate_id\":\"%s\",\"timestamp\":\"%s\"}\n\n",
				event.Type, event.AggregateID(), event.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
			flusher.Flush()
		}
	}
}
