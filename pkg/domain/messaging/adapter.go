// Package messaging defines the pluggable messaging adapter interface.
package messaging

import (
	"context"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
)

// MessageAdapter sends event notifications to an external channel.
type MessageAdapter interface {
	Send(ctx context.Context, event *events.BaseEvent) error
	Name() string
	Type() string
}

// AdapterConfig defines configuration for a messaging adapter.
type AdapterConfig struct {
	Name         string            `yaml:"name" json:"name"`
	Type         string            `yaml:"type" json:"type"` // "webhook", "slack"
	URL          string            `yaml:"url" json:"url"`
	Secret       string            `yaml:"secret,omitempty" json:"secret,omitempty"`
	EventFilters []string          `yaml:"event_filters,omitempty" json:"event_filters,omitempty"`
	Enabled      bool              `yaml:"enabled" json:"enabled"`
	Options      map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
}

// MessagingConfig holds all configured messaging adapters.
type MessagingConfig struct {
	Adapters []AdapterConfig `yaml:"adapters" json:"adapters"`
}
