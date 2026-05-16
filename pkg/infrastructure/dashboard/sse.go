package dashboard

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// sseHub fans state-change notifications out to every connected client.
// Each /events subscriber owns a buffered channel; broadcastChange drops the
// message rather than blocking when a subscriber is slow — the client will
// catch up on the next event or reload.
type sseHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{clients: map[chan string]struct{}{}}
}

func (h *sseHub) subscribe() chan string {
	ch := make(chan string, 4)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *sseHub) unsubscribe(ch chan string) {
	h.mu.Lock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
	h.mu.Unlock()
}

func (h *sseHub) broadcast(event string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
			// drop on slow client; they'll resync on next event or reload
		}
	}
}

// broadcastChange notifies all /events subscribers that the board state has
// mutated. Called from every action handler after a successful transition.
func (s *Server) broadcastChange() {
	if s.sse != nil {
		s.sse.broadcast("task-changed")
	}
}

// handleEvents serves the SSE stream. The client opens an EventSource on
// /events and receives `event: task-changed\ndata: <ts>\n\n` whenever the
// server mutates state. A 25s heartbeat keeps proxies from dropping the
// connection.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx + cloudflare hint

	if s.sse == nil {
		s.sse = newSSEHub()
	}
	ch := s.sse.subscribe()
	defer s.sse.unsubscribe(ch)

	// Initial comment so the client treats the stream as open immediately.
	if _, err := fmt.Fprintf(w, ": connected at %s\n\n", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping %d\n\n", time.Now().Unix()); err != nil {
				return
			}
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %d\n\n", ev, time.Now().UnixNano()); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
