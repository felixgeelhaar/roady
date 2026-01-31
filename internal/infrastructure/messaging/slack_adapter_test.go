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

func TestSlackAdapter_Send(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := messaging.NewSlackAdapter(domainmsg.AdapterConfig{
		Name:    "test-slack",
		Type:    "slack",
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

	if _, ok := payload["text"]; !ok {
		t.Error("expected 'text' field in Slack payload")
	}
}

func TestSlackAdapter_NameAndType(t *testing.T) {
	adapter := messaging.NewSlackAdapter(domainmsg.AdapterConfig{
		Name: "my-slack",
		Type: "slack",
	})

	if adapter.Name() != "my-slack" {
		t.Errorf("expected name 'my-slack', got %q", adapter.Name())
	}
	if adapter.Type() != "slack" {
		t.Errorf("expected type 'slack', got %q", adapter.Type())
	}
}
