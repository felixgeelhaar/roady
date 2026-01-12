package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
	"github.com/spf13/cobra"
	"sort"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show a high-level summary of the project state",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)

		spec, err := repo.LoadSpec()
		if err != nil {
			return err
		}

		plan, err := repo.LoadPlan()
		if err != nil {
			return err
		}

		state, err := repo.LoadState()
		if err != nil {
			return err
		}

		fmt.Printf("Project: %s (v%s)\n", spec.Title, spec.Version)
		fmt.Printf("Spec features: %d\n", len(spec.Features))
		
		if plan == nil {
			fmt.Println("Plan status: No plan generated yet. Run 'roady plan generate'.")
			return nil
		}
		fmt.Printf("Plan status: %s\n", plan.ApprovalStatus)
		
		counts := make(map[planning.TaskStatus]int)
		for _, t := range plan.Tasks {
			status := planning.StatusPending
			if state != nil {
				if res, ok := state.TaskStates[t.ID]; ok {
					status = res.Status
				}
			}
			counts[status]++
		}

		fmt.Printf("Plan tasks: %d\n", len(plan.Tasks))
		fmt.Printf("- ⏳ Pending:     %d\n", counts[planning.StatusPending])
		fmt.Printf("- ✋ Blocked:     %d\n", counts[planning.StatusBlocked])
		fmt.Printf("- 🚧 In Progress: %d\n", counts[planning.StatusInProgress])
		fmt.Printf("- 🏁 Done:        %d (awaiting verification)\n", counts[planning.StatusDone])
		fmt.Printf("- ✅ Verified:    %d\n", counts[planning.StatusVerified])

		if len(plan.Tasks) > 0 {
			totalDone := counts[planning.StatusDone] + counts[planning.StatusVerified]
			progress := float64(totalDone) / float64(len(plan.Tasks)) * 100
			fmt.Printf("\nOverall Progress: %.1f%% (%d/%d tasks finished)\n", progress, totalDone, len(plan.Tasks))
		}

		// List tasks sorted by status
		fmt.Println("\n📋 Task Overview")
		fmt.Println("----------------")
		
		statusRank := map[planning.TaskStatus]int{
			planning.StatusPending:    0,
			planning.StatusBlocked:    1,
			planning.StatusInProgress: 2,
			planning.StatusDone:       3,
			planning.StatusVerified:   4,
		}

		sortedTasks := make([]planning.Task, len(plan.Tasks))
		copy(sortedTasks, plan.Tasks)

		sort.Slice(sortedTasks, func(i, j int) bool {
			sI := planning.StatusPending
			sJ := planning.StatusPending
			if state != nil {
				if res, ok := state.TaskStates[sortedTasks[i].ID]; ok { sI = res.Status }
				if res, ok := state.TaskStates[sortedTasks[j].ID]; ok { sJ = res.Status }
			}
			if sI != sJ {
				return statusRank[sI] < statusRank[sJ]
			}
			return sortedTasks[i].Priority > sortedTasks[j].Priority // Sub-sort by priority
		})

		for _, t := range sortedTasks {
			status := planning.StatusPending
			icon := "⏳"
			if state != nil {
				if res, ok := state.TaskStates[t.ID]; ok {
					status = res.Status
					switch status {
					case planning.StatusVerified: icon = "✅"
					case planning.StatusDone: icon = "🏁"
					case planning.StatusInProgress: icon = "🚧"
					case planning.StatusBlocked: icon = "✋"
					}
				}
			}
			fmt.Printf("%s [%-11s] %-40s (Priority: %s)\n", icon, status, t.Title, t.Priority)
		}

		// Implicit Drift Check

		// Implicit Drift Check
		driftSvc := application.NewDriftService(repo)
		report, err := driftSvc.DetectDrift()
		if err == nil && len(report.Issues) > 0 {
			fmt.Printf("\n⚠️  DRIFT DETECTED: %d issues found. Run 'roady drift detect' for details.\n", len(report.Issues))
		}

		fmt.Printf("\nAudit Trail: .roady/events.jsonl\n")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(statusCmd)
}