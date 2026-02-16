package messaging_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/internal/infrastructure/messaging"
	domainmsg "github.com/felixgeelhaar/roady/pkg/domain/messaging"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
)

func TestWebhookAdapter_Send_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := messaging.NewWebhookAdapter(domainmsg.AdapterConfig{
		Name:    "test-webhook",
		Type:    "webhook",
		URL:     server.URL,
		Enabled: true,
	})

	event := &events.BaseEvent{
		Type:      events.EventTypeTaskCompleted,
		Timestamp: time.Now(),
	}

	err := adapter.Send(context.Background(), event)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	if len(receivedBody) == 0 {
		t.Fatal("expected body to be sent")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := payload["event_type"]; !ok {
		t.Error("expected 'event_type' field in webhook payload")
	}
	if _, ok := payload["timestamp"]; !ok {
		t.Error("expected 'timestamp' field in webhook payload")
	}
	if _, ok := payload["data"]; !ok {
		t.Error("expected 'data' field in webhook payload")
	}
}

func TestWebhookAdapter_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := messaging.NewWebhookAdapter(domainmsg.AdapterConfig{
		Name:    "test-webhook",
		Type:    "webhook",
		URL:     server.URL,
		Enabled: true,
	})

	event := &events.BaseEvent{
		Type:      events.EventTypeTaskCompleted,
		Timestamp: time.Now(),
	}

	err := adapter.Send(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}

	// Verify error message mentions the status code
	expectedMsg := "webhook returned status 500"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestWebhookAdapter_Send_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to ensure context cancellation happens first
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := messaging.NewWebhookAdapter(domainmsg.AdapterConfig{
		Name:    "test-webhook",
		Type:    "webhook",
		URL:     server.URL,
		Enabled: true,
	})

	event := &events.BaseEvent{
		Type:      events.EventTypeTaskCompleted,
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := adapter.Send(ctx, event)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestWebhookAdapter_NameAndType(t *testing.T) {
	adapter := messaging.NewWebhookAdapter(domainmsg.AdapterConfig{
		Name: "my-webhook",
		Type: "webhook",
	})

	if adapter.Name() != "my-webhook" {
		t.Errorf("expected name 'my-webhook', got %q", adapter.Name())
	}
	if adapter.Type() != "webhook" {
		t.Errorf("expected type 'webhook', got %q", adapter.Type())
	}
}
