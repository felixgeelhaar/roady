package cli

import (
	"fmt"
	"math"
	"os"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
	"github.com/spf13/cobra"
)

var forecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Predict project completion based on current velocity",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		auditSvc := application.NewAuditService(repo)
		
		velocity, err := auditSvc.GetVelocity()
		if err != nil {
			return err
		}

		plan, err := repo.LoadPlan()
		if err != nil || plan == nil {
			return fmt.Errorf("no plan found to forecast")
		}

		state, _ := repo.LoadState()
		remaining := 0
		for _, t := range plan.Tasks {
			status := planning.StatusPending
			if state != nil {
				if res, ok := state.TaskStates[t.ID]; ok {
					status = res.Status
				}
			}
			if status != planning.StatusVerified {
				remaining++
			}
		}

		fmt.Println("📈 Project Forecast")
		fmt.Println("------------------")
		fmt.Printf("Current Velocity: %.2f tasks/day\n", velocity)
		fmt.Printf("Remaining Tasks:  %d\n", remaining)

		if remaining == 0 {
			fmt.Println("\nAll tasks verified. Mission complete! 🚀")
			return nil
		}

		if velocity <= 0 {
			fmt.Println("\nUnable to forecast: No velocity data yet. Verify some tasks to enable predictions.")
			return nil
		}

		daysRemaining := math.Ceil(float64(remaining) / velocity)
		fmt.Printf("\nEstimated time to completion: %.0f days\n", daysRemaining)
		
		return nil
	},
}

func init() {
	RootCmd.AddCommand(forecastCmd)
}
