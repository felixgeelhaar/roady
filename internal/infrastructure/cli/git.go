package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git-based automation for Roady",
}

var gitSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Scan recent git commits for task completion markers [roady:task-id]",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		results, err := services.Git.SyncMarkers(10)
		if err != nil {
			return err
		}

		fmt.Printf("Scanning recent commits for Roady markers...\n")
		if len(results) == 0 {
			fmt.Println("No markers found.")
			return nil
		}

		for _, res := range results {
			fmt.Printf("- %s\n", res)
		}

		return nil
	},
}

func init() {
	gitCmd.AddCommand(gitSyncCmd)
	RootCmd.AddCommand(gitCmd)
}
