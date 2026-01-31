package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/messaging"

	msginfra "github.com/felixgeelhaar/roady/internal/infrastructure/messaging"
	"github.com/spf13/cobra"
)

var messagingCmd = &cobra.Command{
	Use:   "messaging",
	Short: "Manage messaging adapters (webhook, Slack, etc.)",
}

var messagingListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured messaging adapters",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		config, err := services.Workspace.Repo.LoadMessagingConfig()
		if err != nil {
			fmt.Println("No messaging adapters configured.")
			return nil
		}

		if len(config.Adapters) == 0 {
			fmt.Println("No messaging adapters configured.")
			return nil
		}

		data, err := json.MarshalIndent(config.Adapters, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

var messagingAddCmd = &cobra.Command{
	Use:   "add <name> <type> <url>",
	Short: "Add a messaging adapter (types: webhook, slack)",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, adapterType, url := args[0], args[1], args[2]

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		repo := services.Workspace.Repo
		config, _ := repo.LoadMessagingConfig()
		if config == nil {
			config = &messaging.MessagingConfig{}
		}

		// Check for duplicate
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

		fmt.Printf("Added %s adapter %q → %s\n", adapterType, name, url)
		return nil
	},
}

var messagingTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Send a test event to a messaging adapter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		config, err := services.Workspace.Repo.LoadMessagingConfig()
		if err != nil {
			return fmt.Errorf("no messaging config found")
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

		testConfig := &messaging.MessagingConfig{
			Adapters: []messaging.AdapterConfig{*target},
		}

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
			if err := adapter.Send(context.Background(), testEvent); err != nil {
				fmt.Printf("Failed to send test to %q: %v\n", adapter.Name(), err)
				return nil
			}
		}

		fmt.Printf("Test event sent to adapter %q\n", name)
		return nil
	},
}

func init() {
	messagingCmd.AddCommand(messagingListCmd)
	messagingCmd.AddCommand(messagingAddCmd)
	messagingCmd.AddCommand(messagingTestCmd)
	RootCmd.AddCommand(messagingCmd)
}
