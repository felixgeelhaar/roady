package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/infrastructure/plugin"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [plugin-path]",
	Short: "Sync the plan with an external system via a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginPath := args[0]
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		loader := plugin.NewLoader()
		defer loader.Cleanup()

		syncer, err := loader.Load(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to load plugin: %w", err)
		}

		plan, err := repo.LoadPlan()
		if err != nil {
			return fmt.Errorf("failed to load plan: %w", err)
		}

		state, err := repo.LoadState()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		if err := syncer.Init(map[string]string{}); err != nil {
			return fmt.Errorf("failed to initialize plugin: %w", err)
		}

		result, err := syncer.Sync(plan, state)
		if err != nil {
			return fmt.Errorf("failed to sync: %w", err)
		}

		audit := application.NewAuditService(repo)
		taskService := application.NewTaskService(repo, audit)

		// 1. Handle Link Updates (New Tickets Created)
		if len(result.LinkUpdates) > 0 {
			fmt.Printf("Received %d link updates from plugin:\n", len(result.LinkUpdates))
			provider := "external" // Default, could be refined
			if strings.Contains(pluginPath, "linear") { provider = "linear" }
			if strings.Contains(pluginPath, "jira") { provider = "jira" }

			for id, ref := range result.LinkUpdates {
				if err := taskService.LinkTask(id, provider, ref); err != nil {
					fmt.Printf("  Failed to link task %s: %v\n", id, err)
				} else {
					fmt.Printf("  Linked task %s to %s (%s)\n", id, provider, ref.Identifier)
				}
			}
		}

		// 2. Handle Status Updates
		updates := result.StatusUpdates
		if len(updates) > 0 {
			fmt.Printf("Received %d status updates from plugin:\n", len(updates))
			for id, status := range updates {
				fmt.Printf("- %s -> %s\n", id, status)
				var event string
				switch status {
				case "done": event = "complete"
				case "in_progress": event = "start"
				}
				
				if event != "" {
					if err := taskService.TransitionTask(id, event, "sync-plugin", ""); err != nil {
						fmt.Printf("  Failed to apply transition: %v\n", err)
					} else {
						fmt.Printf("  Applied.\n")
					}
				}
			}
		}

		if len(result.Errors) > 0 {
			fmt.Printf("\nWarnings/Errors during sync:\n")
			for _, e := range result.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}

		if len(result.LinkUpdates) == 0 && len(result.StatusUpdates) == 0 {
			fmt.Println("No updates from plugin.")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(syncCmd)
}

