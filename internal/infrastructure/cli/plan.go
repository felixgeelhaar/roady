package cli

import (
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage execution plans",
}

var useAI bool

var planGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a plan from the current spec",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		var plan *planning.Plan
		if useAI {
			plan, err = services.AI.DecomposeSpec(cmd.Context())
		} else {
			plan, err = services.Plan.GeneratePlan(cmd.Context())
		}

		if err != nil {
			return MapError(err)
		}

		if plan == nil {
			return fmt.Errorf("plan generation returned no plan and no error")
		}

		fmt.Printf("Successfully generated plan: %s\n", plan.ID)
		fmt.Printf("Status: %s\n", plan.ApprovalStatus)
		fmt.Printf("Tasks generated: %d\n", len(plan.Tasks))

		state, _ := services.Workspace.Repo.LoadState()
		for _, t := range plan.Tasks {
			status := "pending"
			if res, ok := state.TaskStates[t.ID]; ok {
				status = string(res.Status)
			}
			fmt.Printf("- [%s] %s (Priority: %s, Estimate: %s)\n", status, t.Title, t.Priority, t.Estimate)
		}
		return nil
	},
}

var planApproveCmd = &cobra.Command{

	Use: "approve",

	Short: "Approve the current plan for execution",

	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		if err := services.Plan.ApprovePlan(); err != nil {
			return MapError(fmt.Errorf("failed to approve plan: %w", err))
		}

		fmt.Println("Plan approved. You can now start tasks.")
		return nil
	},
}

var planRejectCmd = &cobra.Command{

	Use: "reject",

	Short: "Reject the current plan",

	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		if err := services.Plan.RejectPlan(); err != nil {
			return MapError(fmt.Errorf("failed to reject plan: %w", err))
		}

		fmt.Println("Plan rejected.")
		return nil
	},
}

var planPruneCmd = &cobra.Command{

	Use: "prune",

	Short: "Remove tasks from the plan that are no longer in the spec",

	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		if err := services.Plan.PrunePlan(); err != nil {
			return MapError(fmt.Errorf("failed to prune plan: %w", err))
		}

		fmt.Println("Plan pruned. Orphan tasks removed.")
		return nil
	},
}

var planPrioritizeCmd = &cobra.Command{
	Use:   "prioritize",
	Short: "Get AI-powered priority suggestions for plan tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		if services.AI == nil {
			return MapError(fmt.Errorf("AI service not available; configure an AI provider"))
		}

		suggestions, err := services.AI.SuggestPriorities(cmd.Context())
		if err != nil {
			return MapError(fmt.Errorf("failed to suggest priorities: %w", err))
		}

		fmt.Println("\n--- AI Priority Suggestions ---")
		fmt.Println(suggestions.Summary)
		if len(suggestions.Suggestions) > 0 {
			fmt.Println("\nSuggested changes:")
			for _, s := range suggestions.Suggestions {
				fmt.Printf("  %s: %s → %s\n    %s\n", s.TaskID, s.CurrentPriority, s.SuggestedPriority, s.Reason)
			}
		} else {
			fmt.Println("\nAll task priorities look appropriate.")
		}
		fmt.Println("-------------------------------")
		return nil
	},
}

var planSmartDecomposeCmd = &cobra.Command{
	Use:   "smart-decompose",
	Short: "AI-powered context-aware task decomposition using codebase analysis",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		if services.AI == nil {
			return MapError(fmt.Errorf("AI service not available; configure an AI provider"))
		}

		cwd, cErr := getProjectRoot()
		if cErr != nil {
			return MapError(fmt.Errorf("resolve project path: %w", cErr))
		}

		result, err := services.AI.SmartDecompose(cmd.Context(), cwd)
		if err != nil {
			return MapError(fmt.Errorf("smart decompose failed: %w", err))
		}

		fmt.Println("\n--- Smart Decomposition ---")
		fmt.Println(result.Summary)
		fmt.Printf("\nTasks (%d):\n", len(result.Tasks))
		for _, t := range result.Tasks {
			fmt.Printf("  %-30s [%s] %s\n", t.ID, t.Complexity, t.Title)
			if len(t.Files) > 0 {
				for _, f := range t.Files {
					fmt.Printf("    → %s\n", f)
				}
			}
		}
		fmt.Println("---------------------------")
		return nil
	},
}

func init() {

	planGenerateCmd.Flags().BoolVar(&useAI, "ai", false, "Use AI to decompose the spec into tasks")

	planCmd.AddCommand(planGenerateCmd)

	planCmd.AddCommand(planApproveCmd)

	planCmd.AddCommand(planRejectCmd)

	planCmd.AddCommand(planPruneCmd)

	planCmd.AddCommand(planPrioritizeCmd)

	planCmd.AddCommand(planSmartDecomposeCmd)

	RootCmd.AddCommand(planCmd)

}
