// Package messaging provides pluggable messaging adapter implementations.
package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/messaging"
)

// WebhookAdapter sends events to a generic webhook URL.
type WebhookAdapter struct {
	config messaging.AdapterConfig
	client *http.Client
}

// NewWebhookAdapter creates a webhook adapter from config.
func NewWebhookAdapter(config messaging.AdapterConfig) *WebhookAdapter {
	return &WebhookAdapter{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *WebhookAdapter) Name() string { return a.config.Name }
func (a *WebhookAdapter) Type() string { return "webhook" }

func (a *WebhookAdapter) Send(ctx context.Context, event *events.BaseEvent) error {
	payload := map[string]interface{}{
		"event_type": event.Type,
		"timestamp":  event.Timestamp,
		"data":       event,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Roady-Messaging/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
