package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type mockProcessor struct {
	err    error
	called bool
	event  *Event
}

func (p *mockProcessor) ProcessEvent(_ context.Context, e *Event) error {
	p.called = true
	p.event = e
	return p.err
}

func validGitHubIssuePayload() []byte {
	payload := map[string]interface{}{
		"action": "opened",
		"issue": map[string]interface{}{
			"number":   42,
			"title":    "Test Issue",
			"body":     "roady-id: task-1",
			"state":    "open",
			"html_url": "https://github.com/test/repo/issues/42",
		},
		"repository": map[string]interface{}{
			"full_name": "test/repo",
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func signGitHub(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// --- handleWebhook tests ---

func TestHandleWebhook_MethodNotAllowed(t *testing.T) {
	srv := NewServer(":0", nil)
	srv.RegisterHandler(NewGitHubHandler())
	handler := srv.handleWebhook("github")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/webhooks/github", nil)
	handler(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleWebhook_NoHandler(t *testing.T) {
	srv := NewServer(":0", nil)
	// Don't register any handler
	handler := srv.handleWebhook("github")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", nil)
	handler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleWebhook_InvalidSignature(t *testing.T) {
	srv := NewServer(":0", nil)
	srv.RegisterHandler(NewGitHubHandler())
	srv.SetSecret("github", "correct-secret")
	handler := srv.handleWebhook("github")

	body := validGitHubIssuePayload()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	handler(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleWebhook_ParseError(t *testing.T) {
	srv := NewServer(":0", nil)
	srv.RegisterHandler(NewGitHubHandler())
	handler := srv.handleWebhook("github")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader("not json"))
	r.Header.Set("X-GitHub-Event", "issues")
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleWebhook_ProcessorError(t *testing.T) {
	proc := &mockProcessor{err: errors.New("processing failed")}
	srv := NewServer(":0", proc)
	srv.RegisterHandler(NewGitHubHandler())
	handler := srv.handleWebhook("github")

	body := validGitHubIssuePayload()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "issues")
	handler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if !proc.called {
		t.Error("expected processor to be called")
	}
}

func TestHandleWebhook_Success(t *testing.T) {
	proc := &mockProcessor{}
	srv := NewServer(":0", proc)
	srv.RegisterHandler(NewGitHubHandler())
	handler := srv.handleWebhook("github")

	body := validGitHubIssuePayload()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "issues")
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !proc.called {
		t.Error("expected processor to be called")
	}
	if proc.event.TaskID != "task-1" {
		t.Errorf("expected task ID 'task-1', got %q", proc.event.TaskID)
	}
}

func TestHandleWebhook_SuccessWithSignature(t *testing.T) {
	proc := &mockProcessor{}
	srv := NewServer(":0", proc)
	srv.RegisterHandler(NewGitHubHandler())
	srv.SetSecret("github", "my-secret")
	handler := srv.handleWebhook("github")

	body := validGitHubIssuePayload()
	sig := signGitHub(body, "my-secret")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-Hub-Signature-256", sig)
	r.Header.Set("X-GitHub-Event", "issues")
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleWebhook_NilProcessor(t *testing.T) {
	srv := NewServer(":0", nil)
	srv.RegisterHandler(NewGitHubHandler())
	handler := srv.handleWebhook("github")

	body := validGitHubIssuePayload()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "issues")
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- handleEvents tests ---

func TestHandleEvents_Empty(t *testing.T) {
	srv := NewServer(":0", nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	srv.handleEvents(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var events []Event
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestHandleEvents_WithStoredEvents(t *testing.T) {
	srv := NewServer(":0", nil)

	ev := &Event{
		Provider:  "github",
		EventType: "issues",
		TaskID:    "task-1",
		Status:    planning.StatusPending,
		Timestamp: time.Now(),
	}
	srv.storeEvent(ev)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	srv.handleEvents(w, r)

	var events []Event
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].TaskID != "task-1" {
		t.Errorf("expected task ID 'task-1', got %q", events[0].TaskID)
	}
}

// --- storeEvent tests ---

func TestStoreEvent_Basic(t *testing.T) {
	srv := NewServer(":0", nil)

	ev := &Event{Provider: "github", TaskID: "task-1"}
	srv.storeEvent(ev)

	events := srv.RecentEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].TaskID != "task-1" {
		t.Errorf("expected task ID 'task-1', got %q", events[0].TaskID)
	}
}

func TestStoreEvent_MaxCapacity(t *testing.T) {
	srv := NewServer(":0", nil)

	// Store 105 events
	for i := 0; i < 105; i++ {
		ev := &Event{Provider: "test", TaskID: "task-" + strings.Repeat("x", i)}
		srv.storeEvent(ev)
	}

	events := srv.RecentEvents()
	if len(events) != 100 {
		t.Errorf("expected max 100 events, got %d", len(events))
	}
}

// --- Start/Shutdown integration ---

func TestServer_StartAndShutdown(t *testing.T) {
	proc := &mockProcessor{}
	srv := NewServer("127.0.0.1:0", proc)
	srv.RegisterHandler(NewGitHubHandler())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Give server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Start should return http.ErrServerClosed
	err := <-errCh
	if err != nil && err != http.ErrServerClosed {
		t.Errorf("expected ErrServerClosed, got %v", err)
	}
}
