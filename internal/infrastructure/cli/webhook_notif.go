package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/storage"

	webhook "github.com/felixgeelhaar/roady/internal/infrastructure/webhook"
	"github.com/spf13/cobra"
)

var webhookNotifCmd = &cobra.Command{
	Use:   "notif",
	Short: "Manage outgoing webhook notifications",
}

var webhookNotifAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add an outgoing webhook endpoint",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, url := args[0], args[1]
		secret, _ := cmd.Flags().GetString("secret")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		repo := services.Workspace.Repo
		config, _ := repo.LoadWebhookConfig()
		if config == nil {
			config = &events.WebhookConfig{}
		}

		// Check for duplicate
		for _, ep := range config.Webhooks {
			if ep.Name == name {
				return fmt.Errorf("webhook %q already exists", name)
			}
		}

		config.Webhooks = append(config.Webhooks, events.WebhookEndpoint{
			Name:       name,
			URL:        url,
			Secret:     secret,
			MaxRetries: 3,
			RetryDelay: time.Second,
			Enabled:    true,
		})

		if err := repo.SaveWebhookConfig(config); err != nil {
			return err
		}

		fmt.Printf("Added webhook %q → %s\n", name, url)
		return nil
	},
}

var webhookNotifRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an outgoing webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		repo := services.Workspace.Repo
		config, err := repo.LoadWebhookConfig()
		if err != nil {
			return fmt.Errorf("no webhook config found")
		}

		found := false
		var remaining []events.WebhookEndpoint
		for _, ep := range config.Webhooks {
			if ep.Name == name {
				found = true
				continue
			}
			remaining = append(remaining, ep)
		}

		if !found {
			return fmt.Errorf("webhook %q not found", name)
		}

		config.Webhooks = remaining
		if err := repo.SaveWebhookConfig(config); err != nil {
			return err
		}

		fmt.Printf("Removed webhook %q\n", name)
		return nil
	},
}

var webhookNotifListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured outgoing webhook endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		repo := services.Workspace.Repo
		config, err := repo.LoadWebhookConfig()
		if err != nil {
			fmt.Println("No outgoing webhooks configured.")
			return nil
		}

		if len(config.Webhooks) == 0 {
			fmt.Println("No outgoing webhooks configured.")
			return nil
		}

		for _, ep := range config.Webhooks {
			status := "enabled"
			if !ep.Enabled {
				status = "disabled"
			}
			filters := "all events"
			if len(ep.EventFilters) > 0 {
				data, _ := json.Marshal(ep.EventFilters)
				filters = string(data)
			}
			fmt.Printf("  %s → %s [%s] filters=%s\n", ep.Name, ep.URL, status, filters)
		}
		return nil
	},
}

var webhookNotifTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Send a test event to a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		repo := services.Workspace.Repo
		config, err := repo.LoadWebhookConfig()
		if err != nil {
			return fmt.Errorf("no webhook config found")
		}

		var target *events.WebhookEndpoint
		for i, ep := range config.Webhooks {
			if ep.Name == name {
				target = &config.Webhooks[i]
				break
			}
		}

		if target == nil {
			return fmt.Errorf("webhook %q not found", name)
		}

		dlPath := storage.DeadLetterFile
		dlStore := webhook.NewDeadLetterStore(dlPath)
		notifier := webhook.NewNotifier([]events.WebhookEndpoint{*target}, dlStore)

		testEvent := &events.BaseEvent{
			Type:      "test.ping",
			Timestamp: time.Now(),
			Actor:     "roady-cli",
		}

		notifier.Notify(context.Background(), testEvent)

		// Give async delivery a moment
		time.Sleep(2 * time.Second)
		fmt.Printf("Test event sent to webhook %q\n", name)
		return nil
	},
}

func init() {
	webhookNotifAddCmd.Flags().String("secret", "", "HMAC-SHA256 signing secret")
	webhookNotifCmd.AddCommand(webhookNotifAddCmd)
	webhookNotifCmd.AddCommand(webhookNotifRemoveCmd)
	webhookNotifCmd.AddCommand(webhookNotifListCmd)
	webhookNotifCmd.AddCommand(webhookNotifTestCmd)
	webhookCmd.AddCommand(webhookNotifCmd)
}
