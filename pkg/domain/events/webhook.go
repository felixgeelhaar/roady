package events

import "time"

// WebhookConfig defines configuration for outgoing webhook notifications.
type WebhookConfig struct {
	Webhooks []WebhookEndpoint `yaml:"webhooks" json:"webhooks"`
}

// WebhookEndpoint configures a single outgoing webhook.
type WebhookEndpoint struct {
	Name         string        `yaml:"name" json:"name"`
	URL          string        `yaml:"url" json:"url"`
	Secret       string        `yaml:"secret,omitempty" json:"secret,omitempty"`
	EventFilters []string      `yaml:"event_filters,omitempty" json:"event_filters,omitempty"` // empty = all events
	MaxRetries   int           `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	RetryDelay   time.Duration `yaml:"retry_delay,omitempty" json:"retry_delay,omitempty"`
	Enabled      bool          `yaml:"enabled" json:"enabled"`
}

// DeadLetter records a failed webhook delivery attempt.
type DeadLetter struct {
	Timestamp    time.Time `json:"timestamp"`
	WebhookName  string    `json:"webhook_name"`
	URL          string    `json:"url"`
	EventType    string    `json:"event_type"`
	Payload      string    `json:"payload"`
	Error        string    `json:"error"`
	Attempts     int       `json:"attempts"`
}
