package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	msginfra "github.com/felixgeelhaar/roady/internal/infrastructure/messaging"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/messaging"
)

// notifyList prints the configured outbound messaging adapters as JSON to
// out. Behaviour is identical for both `roady messaging list` and the
// canonical `roady notify list`.
func notifyList(out io.Writer) error {
	services, err := loadServicesForCurrentDir()
	if err != nil {
		return err
	}

	config, err := services.Workspace.Repo.LoadMessagingConfig()
	if err != nil || config == nil || len(config.Adapters) == 0 {
		_, _ = fmt.Fprintln(out, "No notification adapters configured.")
		return nil
	}

	data, err := json.MarshalIndent(config.Adapters, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, string(data))
	return nil
}

// notifyAdd registers a new adapter. adapterType must be one of webhook|slack.
func notifyAdd(out io.Writer, name, adapterType, url string) error {
	services, err := loadServicesForCurrentDir()
	if err != nil {
		return err
	}

	repo := services.Workspace.Repo
	config, _ := repo.LoadMessagingConfig()
	if config == nil {
		config = &messaging.MessagingConfig{}
	}

	for _, a := range config.Adapters {
		if a.Name == name {
			return fmt.Errorf("adapter %q already exists", name)
		}
	}

	config.Adapters = append(config.Adapters, messaging.AdapterConfig{
		Name:    name,
		Type:    adapterType,
		URL:     url,
		Enabled: true,
	})

	if err := repo.SaveMessagingConfig(config); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "Added %s adapter %q -> %s\n", adapterType, name, url)
	return nil
}

// notifyTest dispatches a synthetic test event through the named adapter.
func notifyTest(out io.Writer, name string) error {
	services, err := loadServicesForCurrentDir()
	if err != nil {
		return err
	}

	config, err := services.Workspace.Repo.LoadMessagingConfig()
	if err != nil || config == nil {
		return fmt.Errorf("no notification config found")
	}

	var target *messaging.AdapterConfig
	for i, a := range config.Adapters {
		if a.Name == name {
			target = &config.Adapters[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("adapter %q not found", name)
	}

	testConfig := &messaging.MessagingConfig{Adapters: []messaging.AdapterConfig{*target}}
	registry, err := msginfra.NewRegistry(testConfig)
	if err != nil {
		return fmt.Errorf("create adapter: %w", err)
	}

	testEvent := &events.BaseEvent{
		Type:      "test.ping",
		Timestamp: time.Now(),
		Actor:     "roady-cli",
	}

	for _, adapter := range registry.Adapters() {
		if sendErr := adapter.Send(context.Background(), testEvent); sendErr != nil {
			_, _ = fmt.Fprintf(out, "Failed to send test to %q: %v\n", adapter.Name(), sendErr)
			return nil
		}
	}

	_, _ = fmt.Fprintf(out, "Test event sent to adapter %q\n", name)
	return nil
}

// notifyRemove deletes an adapter from the messaging config. Returns
// ErrNotifyAdapterNotFound when no adapter with that name exists so callers
// can surface a clean message instead of a generic error.
func notifyRemove(out io.Writer, name string) error {
	services, err := loadServicesForCurrentDir()
	if err != nil {
		return err
	}

	repo := services.Workspace.Repo
	config, err := repo.LoadMessagingConfig()
	if err != nil || config == nil {
		return fmt.Errorf("no notification config found")
	}

	idx := -1
	for i, a := range config.Adapters {
		if a.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("adapter %q not found", name)
	}

	config.Adapters = append(config.Adapters[:idx], config.Adapters[idx+1:]...)
	if err := repo.SaveMessagingConfig(config); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "Removed adapter %q\n", name)
	return nil
}
