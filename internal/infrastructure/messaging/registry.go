package messaging

import (
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/domain/messaging"
)

// Registry creates messaging adapters from configuration.
type Registry struct {
	adapters []messaging.MessageAdapter
}

// NewRegistry creates adapters from a MessagingConfig.
func NewRegistry(config *messaging.MessagingConfig) (*Registry, error) {
	if config == nil {
		return &Registry{}, nil
	}

	var adapters []messaging.MessageAdapter
	for _, cfg := range config.Adapters {
		if !cfg.Enabled {
			continue
		}

		adapter, err := createAdapter(cfg)
		if err != nil {
			return nil, fmt.Errorf("create adapter %q: %w", cfg.Name, err)
		}
		adapters = append(adapters, adapter)
	}

	return &Registry{adapters: adapters}, nil
}

// Adapters returns all active adapters.
func (r *Registry) Adapters() []messaging.MessageAdapter {
	return r.adapters
}

func createAdapter(cfg messaging.AdapterConfig) (messaging.MessageAdapter, error) {
	switch cfg.Type {
	case "webhook":
		return NewWebhookAdapter(cfg), nil
	case "slack":
		return NewSlackAdapter(cfg), nil
	default:
		return nil, fmt.Errorf("unknown adapter type: %s", cfg.Type)
	}
}
