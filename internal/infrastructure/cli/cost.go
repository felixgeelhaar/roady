package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Cost tracking and reporting",
}

var costReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a cost report",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		billingSvc := services.Billing

		opts := application.CostReportOpts{
			TaskID: costTaskID,
			Period: costPeriod,
			Format: costFormat,
		}

		report, err := billingSvc.GetCostReport(opts)
		if err != nil {
			return fmt.Errorf("failed to generate cost report: %w", err)
		}

		if len(report.Entries) == 0 {
			fmt.Println("No time entries found.")
			return nil
		}

		switch costFormat {
		case "csv":
			fmt.Println(report.CSV())
		case "json":
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal report: %w", err)
			}
			fmt.Println(string(data))
		case "markdown":
			printMarkdownReport(report)
		default:
			printTextReport(report)
		}

		if costOutput != "" {
			var content string
			switch costFormat {
			case "csv":
				content = report.CSV()
			case "json":
				data, _ := json.MarshalIndent(report, "", "  ")
				content = string(data)
			case "markdown":
				content = generateMarkdownReport(report)
			default:
				content = generateTextReport(report)
			}
			if err := os.WriteFile(costOutput, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write report: %w", err)
			}
			fmt.Printf("\nReport written to: %s\n", costOutput)
		}

		return nil
	},
}

var costBudgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Show budget status",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		billingSvc := services.Billing

		status, err := billingSvc.GetBudgetStatus()
		if err != nil {
			return fmt.Errorf("failed to get budget status: %w", err)
		}

		if status == nil {
			fmt.Println("No budget configured. Set budget_hours in policy.yaml.")
			return nil
		}

		fmt.Printf("Budget Status\n")
		fmt.Printf("=============\n")
		fmt.Printf("Budget:    %.1f hours\n", float64(status.BudgetHours))
		fmt.Printf("Used:      %.2f hours\n", status.UsedHours)
		fmt.Printf("Remaining: %.2f hours\n", status.Remaining)
		fmt.Printf("Used:      %.1f%%\n", status.PercentUsed)

		if status.OverBudget {
			fmt.Printf("\n[WARNING] You are over budget by %.2f hours!\n", -status.Remaining)
		}

		if status.EstimatedHours > 0 {
			fmt.Printf("\nEstimated Cost\n")
			fmt.Printf("--------------\n")
			fmt.Printf("Estimated: %.2f hours ($%.2f)\n", status.EstimatedHours, status.EstimatedCost)
			fmt.Printf("Actual:    %.2f hours ($%.2f)\n", status.UsedHours, status.ActualCost)
			if status.CostVariance > 0 {
				fmt.Printf("Variance:  +$%.2f (over estimate)\n", status.CostVariance)
			} else if status.CostVariance < 0 {
				fmt.Printf("Variance:  -$%.2f (under estimate)\n", -status.CostVariance)
			} else {
				fmt.Printf("Variance:  $0.00 (on target)\n")
			}
			fmt.Printf("Rate:      $%.2f/hr (%s)\n", status.HourlyRate, status.Currency)
			if status.UnestimatedTasks > 0 {
				fmt.Printf("\n[NOTE] %d task(s) have no estimates (%.0f%% coverage)\n",
					status.UnestimatedTasks, status.EstimateCoverage)
			}
		}

		return nil
	},
}

var costTaskID string
var costPeriod string
var costFormat string
var costOutput string

func init() {
	costReportCmd.Flags().StringVarP(&costTaskID, "task", "", "", "Filter by task ID")
	costReportCmd.Flags().StringVarP(&costPeriod, "period", "", "", "Filter by period (e.g., 2026-01)")
	costReportCmd.Flags().StringVarP(&costFormat, "format", "f", "text", "Output format (text, csv, json, markdown)")
	costReportCmd.Flags().StringVarP(&costOutput, "output", "o", "", "Output file path")

	costCmd.AddCommand(costReportCmd)
	costCmd.AddCommand(costBudgetCmd)
	RootCmd.AddCommand(costCmd)
}

func printTextReport(report *billing.CostReport) {
	fmt.Printf("Cost Report (%s)\n", report.Currency)
	if report.TaxName != "" {
		fmt.Printf("Tax: %s (%.1f%%)\n", report.TaxName, report.TaxPercent)
	}
	fmt.Printf("==================\n\n")
	hasTax := report.TaxPercent > 0
	if hasTax {
		fmt.Printf("Task ID       | Title                | Rate       | Hours  | Cost      | Tax       | Total     | Est.Hrs | Est.Cost  | Variance\n")
		fmt.Printf("--------------+----------------------+------------+--------+-----------+-----------+-----------+---------+-----------+----------\n")
		for _, e := range report.Entries {
			title := e.Title
			if len(title) > 20 {
				title = title[:17] + "..."
			}
			fmt.Printf("%-14s| %-20s | %-10s | %6.2f | $%-9.2f | $%-9.2f | $%-9.2f | %7.2f | $%-9.2f | $%.2f\n",
				e.TaskID, title, e.RateName, e.Hours, e.Cost, e.Tax, e.TotalWithTax,
				e.EstimatedHours, e.EstimatedCost, e.CostVariance)
		}
		fmt.Printf("--------------+----------------------+------------+--------+-----------+-----------+-----------+---------+-----------+----------\n")
		fmt.Printf("TOTAL         |                      |            | %6.2f | $%-9.2f | $%-9.2f | $%-9.2f | %7.2f | $%-9.2f | $%.2f\n",
			report.TotalHours, report.TotalCost, report.TotalTax, report.TotalWithTax,
			report.TotalEstimatedHours, report.TotalEstimatedCost, report.TotalCostVariance)
	} else {
		fmt.Printf("Task ID       | Title                | Rate       | Hours  | Cost      | Est.Hrs | Est.Cost  | Variance\n")
		fmt.Printf("--------------+----------------------+------------+--------+-----------+---------+-----------+----------\n")
		for _, e := range report.Entries {
			title := e.Title
			if len(title) > 20 {
				title = title[:17] + "..."
			}
			fmt.Printf("%-14s| %-20s | %-10s | %6.2f | $%-9.2f | %7.2f | $%-9.2f | $%.2f\n",
				e.TaskID, title, e.RateName, e.Hours, e.Cost,
				e.EstimatedHours, e.EstimatedCost, e.CostVariance)
		}
		fmt.Printf("--------------+----------------------+------------+--------+-----------+---------+-----------+----------\n")
		fmt.Printf("TOTAL         |                      |            | %6.2f | $%-9.2f | %7.2f | $%-9.2f | $%.2f\n",
			report.TotalHours, report.TotalCost,
			report.TotalEstimatedHours, report.TotalEstimatedCost, report.TotalCostVariance)
	}
}

func printMarkdownReport(report *billing.CostReport) {
	fmt.Println("# Cost Report")
	fmt.Println()
	fmt.Printf("**Currency:** %s\n", report.Currency)
	if report.TaxName != "" {
		fmt.Printf("**Tax:** %s (%.1f%%)\n", report.TaxName, report.TaxPercent)
	}
	fmt.Println()

	hasTax := report.TaxPercent > 0
	if hasTax {
		fmt.Println("| Task ID | Title | Rate | Hours | Cost | Tax | Total | Est.Hrs | Est.Cost | Variance |")
		fmt.Println("|---------|-------|------|-------|------|-----|-------|---------|----------|----------|")
		for _, e := range report.Entries {
			fmt.Printf("| %s | %s | %s | %.2f | $%.2f | $%.2f | $%.2f | %.2f | $%.2f | $%.2f |\n",
				e.TaskID, e.Title, e.RateName, e.Hours, e.Cost, e.Tax, e.TotalWithTax,
				e.EstimatedHours, e.EstimatedCost, e.CostVariance)
		}
		fmt.Printf("| **TOTAL** | | | **%.2f** | **$%.2f** | **$%.2f** | **$%.2f** | **%.2f** | **$%.2f** | **$%.2f** |\n",
			report.TotalHours, report.TotalCost, report.TotalTax, report.TotalWithTax,
			report.TotalEstimatedHours, report.TotalEstimatedCost, report.TotalCostVariance)
	} else {
		fmt.Println("| Task ID | Title | Rate | Hours | Cost | Est.Hrs | Est.Cost | Variance |")
		fmt.Println("|---------|-------|------|-------|------|---------|----------|----------|")
		for _, e := range report.Entries {
			fmt.Printf("| %s | %s | %s | %.2f | $%.2f | %.2f | $%.2f | $%.2f |\n",
				e.TaskID, e.Title, e.RateName, e.Hours, e.Cost,
				e.EstimatedHours, e.EstimatedCost, e.CostVariance)
		}
		fmt.Printf("| **TOTAL** | | | **%.2f** | **$%.2f** | **%.2f** | **$%.2f** | **$%.2f** |\n",
			report.TotalHours, report.TotalCost,
			report.TotalEstimatedHours, report.TotalEstimatedCost, report.TotalCostVariance)
	}
}

func generateMarkdownReport(report *billing.CostReport) string {
	return fmt.Sprintf(`# Cost Report

**Currency:** %s
**Generated:** %s

## Summary

- Total Hours: %.2f
- Total Cost: $%.2f
- Estimated Hours: %.2f
- Estimated Cost: $%.2f
- Cost Variance: $%.2f

## Entries

| Task ID | Title | Rate | Hours | Cost | Est.Hrs | Est.Cost | Variance |
|---------|-------|------|-------|------|---------|----------|----------|
%s
`,
		report.Currency,
		report.GeneratedAt.Format("2006-01-02 15:04"),
		report.TotalHours,
		report.TotalCost,
		report.TotalEstimatedHours,
		report.TotalEstimatedCost,
		report.TotalCostVariance,
		generateMarkdownTable(report.Entries),
	)
}

func generateMarkdownTable(entries []billing.CostReportEntry) string {
	result := ""
	for _, e := range entries {
		result += fmt.Sprintf("| %s | %s | %s | %.2f | $%.2f | %.2f | $%.2f | $%.2f |\n",
			e.TaskID, e.Title, e.RateName, e.Hours, e.Cost,
			e.EstimatedHours, e.EstimatedCost, e.CostVariance)
	}
	return result
}

func generateTextReport(report *billing.CostReport) string {
	return fmt.Sprintf(`Cost Report (%s)
==================

Task ID       | Title                | Rate       | Hours  | Cost      | Est.Hrs | Est.Cost  | Variance
--------------+----------------------+------------+--------+-----------+---------+-----------+----------
%s
--------------+----------------------+------------+--------+-----------+---------+-----------+----------
TOTAL         |                      |            | %6.2f | $%-9.2f | %7.2f | $%-9.2f | $%.2f
`,
		report.Currency,
		generateTextTable(report.Entries),
		report.TotalHours,
		report.TotalCost,
		report.TotalEstimatedHours,
		report.TotalEstimatedCost,
		report.TotalCostVariance,
	)
}

func generateTextTable(entries []billing.CostReportEntry) string {
	result := ""
	for _, e := range entries {
		title := e.Title
		if len(title) > 20 {
			title = title[:17] + "..."
		}
		result += fmt.Sprintf("%-14s| %-20s | %-10s | %6.2f | $%-9.2f | %7.2f | $%-9.2f | $%.2f\n",
			e.TaskID, title, e.RateName, e.Hours, e.Cost,
			e.EstimatedHours, e.EstimatedCost, e.CostVariance)
	}
	return result
}
