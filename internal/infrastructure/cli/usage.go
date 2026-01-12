package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show project usage and AI token statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		service := application.NewPlanService(repo, application.NewAuditService(repo))

		stats, err := service.GetUsage()
		if err != nil {
			return fmt.Errorf("failed to load usage stats: %w", err)
		}

		fmt.Println("Project Usage Metrics")
		fmt.Println("-----------------------")
		fmt.Printf("Total Commands: %d\n", stats.TotalCommands)
		if !stats.LastCommandAt.IsZero() {
			fmt.Printf("Last Activity:  %s\n", stats.LastCommandAt.Format("2006-01-02 15:04:05"))
		}

		if len(stats.ProviderStats) > 0 {
			fmt.Println("\nAI Token Consumption")
			
			// Sort keys for stable output
			keys := make([]string, 0, len(stats.ProviderStats))
			for k := range stats.ProviderStats {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Printf("- %-25s: %d\n", k, stats.ProviderStats[k])
			}
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(usageCmd)
}
