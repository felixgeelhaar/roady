package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [plugin-path]",
	Short: "Sync the plan with an external system via a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginPath := args[0]
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		results, err := services.Sync.SyncWithPlugin(pluginPath)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("No updates from plugin.")
			return nil
		}

		fmt.Printf("Sync results for %s:\n", pluginPath)
		for _, res := range results {
			fmt.Printf("- %s\n", res)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(syncCmd)
}
