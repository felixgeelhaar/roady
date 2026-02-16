package cli

import (
	"fmt"
	"sort"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show project usage and AI token statistics",
	RunE:  runUsage,
}

func runUsage(cmd *cobra.Command, args []string) error {
	cwd, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("resolve project path: %w", err)
	}
	workspace := wiring.NewWorkspace(cwd)
	usageService := application.NewUsageService(workspace.Repo)

	stats, err := usageService.GetUsage()
	if err != nil {
		return fmt.Errorf("failed to load usage stats: %w", err)
	}

	fmt.Println("Project Usage Metrics")
	fmt.Println("-----------------------")
	fmt.Printf("Total Commands: %d\n", stats.TotalCommands)
	if !stats.LastCommandAt.IsZero() {
		fmt.Printf("Last Activity:  %s\n", stats.LastCommandAt.Format("2006-01-02 15:04:05"))
	}

	// Calculate total tokens and show budget alerts
	totalTokens := 0
	if len(stats.ProviderStats) > 0 {
		fmt.Println("\nAI Token Consumption")

		// Sort keys for stable output
		keys := make([]string, 0, len(stats.ProviderStats))
		for k := range stats.ProviderStats {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			tokens := stats.ProviderStats[k]
			totalTokens += tokens
			fmt.Printf("- %-25s: %d\n", k, tokens)
		}
		fmt.Printf("\nTotal Tokens Used: %d\n", totalTokens)
	}

	// Check policy token limits and show budget alerts
	policy, policyErr := workspace.Repo.LoadPolicy()
	if policyErr == nil && policy != nil && policy.TokenLimit > 0 {
		limit := policy.TokenLimit
		usagePercent := float64(totalTokens) / float64(limit) * 100

		fmt.Println("\nBudget Status")
		fmt.Printf("Token Limit:    %d\n", limit)
		fmt.Printf("Usage:          %.1f%%\n", usagePercent)

		// Alert thresholds
		switch {
		case usagePercent >= 100:
			fmt.Println("\n[CRITICAL] Token budget EXCEEDED! AI operations may be blocked.")
			fmt.Println("           Consider increasing token_limit in policy.yaml or resetting usage.")
		case usagePercent >= 90:
			fmt.Println("\n[WARNING] Token budget at 90%+. Approaching limit.")
			fmt.Println("          Consider reviewing AI usage or increasing budget.")
		case usagePercent >= 75:
			fmt.Println("\n[INFO] Token budget at 75%+. Monitor usage.")
		}
	}

	return nil
}

// RunUsage is the exported RunE function for use as a subcommand
var RunUsage = runUsage

func init() {
	usageCmd.Hidden = true // Hide from top-level help, available via `status usage`
	RootCmd.AddCommand(usageCmd)
}
