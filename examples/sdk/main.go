// Package main demonstrates how to use Roady as an SDK.
// This example shows programmatic access to Roady's planning services,
// which is the same approach used by both CLI and MCP tools.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
)

func main() {
	// Use current directory as workspace, or specify a path
	workspacePath := "."
	if len(os.Args) > 1 {
		workspacePath = os.Args[1]
	}

	// Build all application services
	// This is the same wiring used by CLI and MCP
	services, err := wiring.BuildAppServices(workspacePath)
	if err != nil {
		log.Fatalf("Failed to build services: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: Get current spec
	fmt.Println("=== Product Specification ===")
	spec, err := services.Spec.GetSpec()
	if err != nil {
		log.Printf("Warning: No spec found: %v", err)
	} else {
		fmt.Printf("Project: %s\n", spec.Title)
		fmt.Printf("Features: %d\n", len(spec.Features))
		for _, f := range spec.Features {
			fmt.Printf("  - %s: %s\n", f.ID, f.Title)
		}
	}

	// Example 2: Get current plan
	fmt.Println("\n=== Execution Plan ===")
	plan, err := services.Plan.GetPlan()
	if err != nil {
		log.Printf("Warning: No plan found: %v", err)
	} else {
		fmt.Printf("Plan ID: %s\n", plan.ID)
		fmt.Printf("Tasks: %d\n", len(plan.Tasks))
		fmt.Printf("Approved: %v\n", plan.ApprovalStatus.IsApproved())
	}

	// Example 3: Get project snapshot (consistent view)
	fmt.Println("\n=== Project Snapshot ===")
	snapshot, err := services.Plan.GetProjectSnapshot(ctx)
	if err != nil {
		log.Printf("Warning: Could not get snapshot: %v", err)
	} else if snapshot.Plan != nil {
		fmt.Printf("Progress: %.1f%%\n", snapshot.Progress*100)
		fmt.Printf("Ready Tasks: %d\n", len(snapshot.UnlockedTasks))
		fmt.Printf("Blocked Tasks: %d\n", len(snapshot.BlockedTasks))
		fmt.Printf("In-Progress Tasks: %d\n", len(snapshot.InProgress))
	}

	// Example 4: Query tasks by status
	fmt.Println("\n=== Task Queries ===")

	readyTasks, err := services.Plan.GetReadyTasks(ctx)
	if err == nil {
		fmt.Printf("Ready to start:\n")
		for _, t := range readyTasks {
			fmt.Printf("  - %s: %s\n", t.ID, t.Title)
		}
	}

	blockedTasks, err := services.Plan.GetBlockedTasks(ctx)
	if err == nil && len(blockedTasks) > 0 {
		fmt.Printf("Blocked:\n")
		for _, t := range blockedTasks {
			fmt.Printf("  - %s: %s\n", t.ID, t.Title)
		}
	}

	inProgressTasks, err := services.Plan.GetInProgressTasks(ctx)
	if err == nil && len(inProgressTasks) > 0 {
		fmt.Printf("In progress:\n")
		for _, t := range inProgressTasks {
			fmt.Printf("  - %s: %s (owner: %s)\n", t.ID, t.Title, t.Owner)
		}
	}

	// Example 5: Check drift
	fmt.Println("\n=== Drift Detection ===")
	driftReport, err := services.Drift.DetectDrift(ctx)
	if err != nil {
		log.Printf("Warning: Drift detection failed: %v", err)
	} else {
		if len(driftReport.Issues) == 0 {
			fmt.Println("No drift detected - project is aligned!")
		} else {
			fmt.Printf("Found %d drift issues:\n", len(driftReport.Issues))
			for _, issue := range driftReport.Issues {
				fmt.Printf("  - [%s] %s: %s\n", issue.Severity, issue.Type, issue.Message)
			}
		}
	}

	// Example 6: Check policy compliance
	fmt.Println("\n=== Policy Compliance ===")
	violations, err := services.Policy.CheckCompliance()
	switch {
	case err != nil:
		log.Printf("Warning: Policy check failed: %v", err)
	case len(violations) == 0:
		fmt.Println("All policies satisfied!")
	default:
		fmt.Printf("Policy violations:\n")
		for _, v := range violations {
			fmt.Printf("  - [%s] %s: %s\n", v.Level, v.RuleID, v.Message)
		}
	}

	// Example 7: Get usage statistics
	fmt.Println("\n=== Usage Statistics ===")
	usage, err := services.Plan.GetUsage()
	if err != nil {
		log.Printf("Warning: Could not get usage: %v", err)
	} else {
		fmt.Printf("Total Commands: %d\n", usage.TotalCommands)
		if len(usage.ProviderStats) > 0 {
			totalTokens := 0
			for provider, tokens := range usage.ProviderStats {
				fmt.Printf("  %s: %d tokens\n", provider, tokens)
				totalTokens += tokens
			}
			fmt.Printf("Total Tokens: %d\n", totalTokens)
		}
	}

	// Example 8: Get velocity forecast
	fmt.Println("\n=== Velocity Forecast ===")
	forecast, err := services.Forecast.GetForecast()
	if err != nil {
		log.Printf("Warning: Could not get forecast: %v", err)
	} else {
		fmt.Printf("Remaining Tasks: %d\n", forecast.RemainingTasks)
		fmt.Printf("Current Velocity: %.2f tasks/day\n", forecast.Velocity)
		if forecast.Velocity > 0 {
			fmt.Printf("Estimated Days to Complete: %.1f\n", forecast.EstimatedDays)
			fmt.Printf("Confidence: [%.1f - %.1f - %.1f] days\n",
				forecast.ConfidenceInterval.Low,
				forecast.ConfidenceInterval.Expected,
				forecast.ConfidenceInterval.High)
		}
		fmt.Printf("Trend Direction: %s\n", forecast.Trend.Direction)
	}

	// Example 9: Get dependency graph
	fmt.Println("\n=== Repository Dependencies ===")
	deps, err := services.Dependency.ListDependencies()
	switch {
	case err != nil:
		log.Printf("Warning: Could not list dependencies: %v", err)
	case len(deps) == 0:
		fmt.Println("No cross-repo dependencies defined")
	default:
		fmt.Printf("Dependencies: %d\n", len(deps))
		for _, d := range deps {
			fmt.Printf("  - %s -> %s (%s)\n", d.SourceRepo, d.TargetRepo, d.Type)
		}
	}

	// Example 10: Get planning debt analysis
	fmt.Println("\n=== Planning Debt ===")
	debtReport, err := services.Debt.GetDebtReport(ctx)
	if err != nil {
		log.Printf("Warning: Could not get debt report: %v", err)
	} else {
		fmt.Printf("Total Items: %d\n", debtReport.TotalItems)
		fmt.Printf("Sticky Items: %d\n", debtReport.StickyItems)
		fmt.Printf("Average Score: %.1f\n", debtReport.AverageScore)
		fmt.Printf("Health Level: %s\n", debtReport.GetHealthLevel())

		// Show top debtors
		topDebtors := debtReport.GetTopDebtors(3)
		if len(topDebtors) > 0 {
			fmt.Printf("Top Debt Components:\n")
			for _, score := range topDebtors {
				fmt.Printf("  - %s: score %.1f (%d items)\n",
					score.ComponentID, score.Score, len(score.Items))
			}
		}

		// Show sticky drift
		if len(debtReport.StickyDrift) > 0 {
			fmt.Printf("Sticky Drift (unresolved >7 days):\n")
			for _, item := range debtReport.StickyDrift {
				fmt.Printf("  - %s: %d days pending (%s)\n",
					item.ComponentID, item.DaysPending, item.DriftType)
			}
		}
	}

	// Example 11: Generate plan (if needed)
	// Uncomment to generate a new plan from the spec
	/*
	fmt.Println("\n=== Generating Plan ===")
	newPlan, err := services.Plan.GeneratePlan(ctx)
	if err != nil {
		log.Fatalf("Failed to generate plan: %v", err)
	}
	fmt.Printf("Generated plan with %d tasks\n", len(newPlan.Tasks))
	*/

	// Example 12: Approve plan (if needed)
	// Uncomment to approve the current plan
	/*
	fmt.Println("\n=== Approving Plan ===")
	err = services.Plan.ApprovePlan()
	if err != nil {
		log.Fatalf("Failed to approve plan: %v", err)
	}
	fmt.Println("Plan approved successfully!")
	*/

	// Example 13: Task transitions
	// Uncomment to start/complete tasks
	/*
	taskID := "task-example"
	err = services.Task.TransitionTask(taskID, "start", "sdk-user", "")
	if err != nil {
		log.Printf("Failed to start task: %v", err)
	}
	err = services.Task.TransitionTask(taskID, "complete", "sdk-user", "commit-123")
	if err != nil {
		log.Printf("Failed to complete task: %v", err)
	}
	*/

	fmt.Println("\n=== SDK Example Complete ===")
}
