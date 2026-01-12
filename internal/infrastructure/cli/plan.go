package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/infrastructure/ai"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
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
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		audit := application.NewAuditService(repo)

		var plan *planning.Plan
		var err error

		if useAI {
			cfg, _ := repo.LoadPolicy()
			pName, mName := "ollama", "llama3"
			if cfg != nil {
				pName = cfg.AIProvider
				mName = cfg.AIModel
			}

			baseProvider, err := ai.GetDefaultProvider(pName, mName)
			if err != nil {
				return err
			}
			provider := ai.NewResilientProvider(baseProvider)
			service := application.NewAIPlanningService(repo, provider, audit)
			plan, err = service.DecomposeSpec(cmd.Context())
		} else {
			service := application.NewPlanService(repo, audit)
			plan, err = service.GeneratePlan()
		}

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}

		if plan == nil {
			return fmt.Errorf("plan generation returned no plan and no error")
		}

		fmt.Printf("Successfully generated plan: %s\n", plan.ID)
		fmt.Printf("Status: %s\n", plan.ApprovalStatus)
		fmt.Printf("Tasks generated: %d\n", len(plan.Tasks))

		state, _ := repo.LoadState()
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

	Use:   "approve",

	Short: "Approve the current plan for execution",

	RunE: func(cmd *cobra.Command, args []string) error {

		cwd, _ := os.Getwd()

		repo := storage.NewFilesystemRepository(cwd)

		audit := application.NewAuditService(repo)

		service := application.NewPlanService(repo, audit)



		if err := service.ApprovePlan(); err != nil {

			return fmt.Errorf("failed to approve plan: %w", err)

		}

		fmt.Println("Plan approved. You can now start tasks.")

		return nil

	},

}



var planRejectCmd = &cobra.Command{

	Use:   "reject",

	Short: "Reject the current plan",

	RunE: func(cmd *cobra.Command, args []string) error {

		cwd, _ := os.Getwd()

		repo := storage.NewFilesystemRepository(cwd)

		audit := application.NewAuditService(repo)

		service := application.NewPlanService(repo, audit)



		if err := service.RejectPlan(); err != nil {

			return fmt.Errorf("failed to reject plan: %w", err)

		}

		fmt.Println("Plan rejected.")

		return nil

	},

}



var planPruneCmd = &cobra.Command{



	Use:   "prune",



	Short: "Remove tasks from the plan that are no longer in the spec",



	RunE: func(cmd *cobra.Command, args []string) error {



		cwd, _ := os.Getwd()



		repo := storage.NewFilesystemRepository(cwd)



		audit := application.NewAuditService(repo)



		service := application.NewPlanService(repo, audit)







		if err := service.PrunePlan(); err != nil {



			return fmt.Errorf("failed to prune plan: %w", err)



		}



		fmt.Println("Plan pruned. Orphan tasks removed.")



		return nil



	},



}







func init() {



	planGenerateCmd.Flags().BoolVar(&useAI, "ai", false, "Use AI to decompose the spec into tasks")



	planCmd.AddCommand(planGenerateCmd)



	planCmd.AddCommand(planApproveCmd)



	planCmd.AddCommand(planRejectCmd)



	planCmd.AddCommand(planPruneCmd)



	RootCmd.AddCommand(planCmd)



}




