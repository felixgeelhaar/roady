package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var debtCmd = &cobra.Command{
	Use:   "debt",
	Short: "Analyze planning debt and recurring drift patterns",
}

var debtReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate comprehensive debt report",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")
		ctx := context.Background()

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		report, err := services.Debt.GetDebtReport(ctx)
		if err != nil {
			return fmt.Errorf("failed to get debt report: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Debt Report\n")
		fmt.Printf("===========\n\n")
		fmt.Printf("Total Items:    %d\n", report.TotalItems)
		fmt.Printf("Sticky Items:   %d\n", report.StickyItems)
		fmt.Printf("Average Score:  %.2f\n", report.AverageScore)
		fmt.Printf("Health Level:   %s\n", report.GetHealthLevel())
		fmt.Println()

		if len(report.ByCategory) > 0 {
			fmt.Println("By Category:")
			for category, items := range report.ByCategory {
				fmt.Printf("  %s: %d items\n", category, len(items))
			}
			fmt.Println()
		}

		topDebtors := report.GetTopDebtors(5)
		if len(topDebtors) > 0 {
			fmt.Println("Top Debtors:")
			for i, score := range topDebtors {
				fmt.Printf("  %d. %s (score: %.2f, items: %d)\n",
					i+1, score.ComponentID, score.Score, len(score.Items))
			}
			fmt.Println()
		}

		if len(report.RecentlyResolved) > 0 {
			fmt.Printf("Recently Resolved: %d items\n", len(report.RecentlyResolved))
		}

		return nil
	},
}

var debtScoreCmd = &cobra.Command{
	Use:   "score <component-id>",
	Short: "Get debt score for a specific component",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		componentID := args[0]
		outputFormat, _ := cmd.Flags().GetString("output")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		score, err := services.Debt.GetDebtScore(componentID)
		if err != nil {
			return fmt.Errorf("failed to get debt score: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(score, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Debt Score: %s\n", componentID)
		fmt.Printf("=============\n\n")
		fmt.Printf("Score:            %.2f\n", score.Score)
		fmt.Printf("Total Items:      %d\n", len(score.Items))
		fmt.Printf("Sticky Items:     %d\n", score.StickyCount)
		fmt.Printf("Total Days Pending: %d\n", score.TotalDaysPending)
		fmt.Println()

		if len(score.Items) > 0 {
			fmt.Println("Debt Items:")
			for _, item := range score.Items {
				sticky := ""
				if item.IsSticky {
					sticky = " [STICKY]"
				}
				fmt.Printf("  - %s (%s)%s\n", item.DriftType, item.Category, sticky)
				fmt.Printf("    Detected: %d times, pending %d days\n",
					item.DetectionCount, item.DaysPending)
				if item.Message != "" {
					fmt.Printf("    Message: %s\n", item.Message)
				}
			}
		}

		return nil
	},
}

var debtStickyCmd = &cobra.Command{
	Use:   "sticky",
	Short: "Show sticky debt items (unresolved >7 days)",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		items, err := services.Debt.GetStickyDrift()
		if err != nil {
			return fmt.Errorf("failed to get sticky drift: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(items, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(items) == 0 {
			fmt.Println("No sticky debt items found.")
			fmt.Println("Sticky items are drift issues unresolved for more than 7 days.")
			return nil
		}

		fmt.Printf("Sticky Debt Items (%d)\n", len(items))
		fmt.Printf("=====================\n\n")

		for _, item := range items {
			fmt.Printf("Component: %s\n", item.ComponentID)
			fmt.Printf("  Type:     %s\n", item.DriftType)
			fmt.Printf("  Category: %s\n", item.Category)
			fmt.Printf("  Pending:  %d days\n", item.DaysPending)
			fmt.Printf("  Detected: %d times\n", item.DetectionCount)
			if item.Message != "" {
				fmt.Printf("  Message:  %s\n", item.Message)
			}
			fmt.Println()
		}

		return nil
	},
}

var debtSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Quick overview of debt status",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")
		ctx := context.Background()

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		summary, err := services.Debt.GetDebtSummary(ctx)
		if err != nil {
			return fmt.Errorf("failed to get debt summary: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(summary, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Debt Summary\n")
		fmt.Printf("============\n\n")
		fmt.Printf("Health:       %s\n", summary.HealthLevel)
		fmt.Printf("Total Items:  %d\n", summary.TotalItems)
		fmt.Printf("Sticky Items: %d\n", summary.StickyItems)
		fmt.Printf("Avg Score:    %.2f\n", summary.AverageScore)

		if summary.TopDebtor != "" {
			fmt.Printf("\nTop Debtor:   %s (score: %.2f)\n", summary.TopDebtor, summary.TopDebtorScore)
		}

		return nil
	},
}

var debtHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show drift detection history",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")
		days, _ := cmd.Flags().GetInt("days")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		history, err := services.Debt.GetDriftHistory(days)
		if err != nil {
			return fmt.Errorf("failed to get drift history: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(history, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(history) == 0 {
			fmt.Println("No drift history found.")
			if days > 0 {
				fmt.Printf("No drift detected in the last %d days.\n", days)
			}
			return nil
		}

		windowDesc := "all time"
		if days > 0 {
			windowDesc = fmt.Sprintf("last %d days", days)
		}

		fmt.Printf("Drift History (%s)\n", windowDesc)
		fmt.Printf("================\n\n")

		for _, snapshot := range history {
			fmt.Printf("%s\n", snapshot.Timestamp.Format(time.RFC3339))
			if snapshot.ComponentID != "" {
				fmt.Printf("  Component: %s\n", snapshot.ComponentID)
			}
			fmt.Printf("  Type:      %s\n", snapshot.DriftType)
			fmt.Printf("  Category:  %s\n", snapshot.Category)
			if snapshot.Message != "" {
				fmt.Printf("  Message:   %s\n", snapshot.Message)
			}
			fmt.Printf("  Issues:    %d\n", snapshot.IssueCount)
			fmt.Println()
		}

		return nil
	},
}

var debtTrendCmd = &cobra.Command{
	Use:   "trend",
	Short: "Analyze drift trend over time",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")
		days, _ := cmd.Flags().GetInt("days")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		trend, err := services.Debt.GetDriftTrend(days)
		if err != nil {
			return fmt.Errorf("failed to get drift trend: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(trend, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Drift Trend Analysis\n")
		fmt.Printf("====================\n\n")
		fmt.Printf("Window:         %d days\n", trend.WindowDays)
		fmt.Printf("Direction:      %s\n", trend.Direction)
		fmt.Printf("Change:         %.1f%%\n", trend.Change*100)
		fmt.Printf("Previous Period: %d issues\n", trend.PreviousPeriod)
		fmt.Printf("Current Period:  %d issues\n", trend.CurrentPeriod)
		fmt.Println()
		fmt.Printf("Active Debt:    %d items\n", trend.ActiveDebtCount)
		fmt.Printf("Sticky Count:   %d items\n", trend.StickyCount)

		// Add interpretation
		fmt.Println()
		switch trend.Direction {
		case "increasing":
			fmt.Println("Warning: Drift is increasing. Consider addressing root causes.")
		case "decreasing":
			fmt.Println("Good: Drift is decreasing. Keep up the good work!")
		case "stable":
			fmt.Println("Info: Drift levels are stable.")
		}

		return nil
	},
}

func init() {
	// Report command flags
	debtReportCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// Score command flags
	debtScoreCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// Sticky command flags
	debtStickyCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// Summary command flags
	debtSummaryCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// History command flags
	debtHistoryCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	debtHistoryCmd.Flags().IntP("days", "d", 0, "Limit to last N days (0 for all)")

	// Trend command flags
	debtTrendCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	debtTrendCmd.Flags().IntP("days", "d", 30, "Analysis window in days")

	// Add subcommands
	debtCmd.AddCommand(debtReportCmd)
	debtCmd.AddCommand(debtScoreCmd)
	debtCmd.AddCommand(debtStickyCmd)
	debtCmd.AddCommand(debtSummaryCmd)
	debtCmd.AddCommand(debtHistoryCmd)
	debtCmd.AddCommand(debtTrendCmd)

	RootCmd.AddCommand(debtCmd)
}
