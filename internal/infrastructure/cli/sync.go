package cli

import (
	"fmt"
	"os"

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

		updates, err := syncer.Sync(plan, state)
		if err != nil {
			return fmt.Errorf("failed to sync: %w", err)
		}

		if len(updates) > 0 {
			audit := application.NewAuditService(repo)
			taskService := application.NewTaskService(repo, audit)
			fmt.Printf("Received %d updates from plugin:\n", len(updates))
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
		} else {
			fmt.Println("No updates from plugin.")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(syncCmd)
}
