package messaging_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/messaging"
	domainmsg "github.com/felixgeelhaar/roady/pkg/domain/messaging"
)

func TestRegistry_CreatesAdapters(t *testing.T) {
	config := &domainmsg.MessagingConfig{
		Adapters: []domainmsg.AdapterConfig{
			{Name: "webhook1", Type: "webhook", URL: "http://example.com", Enabled: true},
			{Name: "slack1", Type: "slack", URL: "http://slack.com/hook", Enabled: true},
			{Name: "disabled", Type: "webhook", URL: "http://disabled.com", Enabled: false},
		},
	}

	registry, err := messaging.NewRegistry(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	adapters := registry.Adapters()
	if len(adapters) != 2 {
		t.Errorf("expected 2 enabled adapters, got %d", len(adapters))
	}
}

func TestRegistry_UnknownType(t *testing.T) {
	config := &domainmsg.MessagingConfig{
		Adapters: []domainmsg.AdapterConfig{
			{Name: "bad", Type: "unknown", URL: "http://example.com", Enabled: true},
		},
	}

	_, err := messaging.NewRegistry(config)
	if err == nil {
		t.Error("expected error for unknown adapter type")
	}
}

func TestRegistry_NilConfig(t *testing.T) {
	registry, err := messaging.NewRegistry(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(registry.Adapters()) != 0 {
		t.Errorf("expected 0 adapters for nil config")
	}
}
