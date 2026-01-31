// Package webhook provides outgoing webhook notification delivery.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
)

// Notifier sends outgoing webhook notifications for domain events.
type Notifier struct {
	endpoints  []events.WebhookEndpoint
	client     *http.Client
	deadLetter *DeadLetterStore
}

// NewNotifier creates a notifier with the given endpoints and dead letter store.
func NewNotifier(endpoints []events.WebhookEndpoint, deadLetter *DeadLetterStore) *Notifier {
	return &Notifier{
		endpoints: endpoints,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		deadLetter: deadLetter,
	}
}

// Payload is the JSON body sent to webhook endpoints.
type Payload struct {
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Notify sends an event to all matching webhook endpoints.
func (n *Notifier) Notify(ctx context.Context, event *events.BaseEvent) {
	payload := Payload{
		EventType: event.Type,
		Timestamp: event.Timestamp,
		Data:      event,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, ep := range n.endpoints {
		if !ep.Enabled {
			continue
		}
		if !n.matchesFilter(ep, event.Type) {
			continue
		}
		go n.deliver(ctx, ep, event.Type, body)
	}
}

func (n *Notifier) matchesFilter(ep events.WebhookEndpoint, eventType string) bool {
	if len(ep.EventFilters) == 0 {
		return true
	}
	for _, f := range ep.EventFilters {
		if f == eventType {
			return true
		}
	}
	return false
}

func (n *Notifier) deliver(ctx context.Context, ep events.WebhookEndpoint, eventType string, body []byte) {
	maxRetries := ep.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	retryDelay := ep.RetryDelay
	if retryDelay <= 0 {
		retryDelay = time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := n.send(ctx, ep, body); err != nil {
			lastErr = err
			if attempt < maxRetries {
				time.Sleep(retryDelay * time.Duration(attempt)) // linear backoff
			}
			continue
		}
		return // success
	}

	// All retries exhausted — dead letter
	if n.deadLetter != nil && lastErr != nil {
		dl := events.DeadLetter{
			Timestamp:   time.Now(),
			WebhookName: ep.Name,
			URL:         ep.URL,
			EventType:   eventType,
			Payload:     string(body),
			Error:       lastErr.Error(),
			Attempts:    maxRetries,
		}
		_ = n.deadLetter.Append(dl)
	}
}

func (n *Notifier) send(ctx context.Context, ep events.WebhookEndpoint, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Roady-Webhook/1.0")

	if ep.Secret != "" {
		sig := sign(body, ep.Secret)
		req.Header.Set("X-Roady-Signature", sig)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// sign computes HMAC-SHA256 of the payload using the secret.
func sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
