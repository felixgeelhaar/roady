package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between specs, plans, and code",
}

var driftExplainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Provide an AI-generated explanation and resolution steps for current drift",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		driftSvc := application.NewDriftService(repo)
		audit := workspace.Audit

		provider, err := wiring.LoadAIProvider(cwd)
		if err != nil {
			return err
		}
		aiSvc := application.NewAIPlanningService(repo, provider, audit)

		report, err := driftSvc.DetectDrift()
		if err != nil {
			return fmt.Errorf("failed to detect drift: %w", err)
		}

		explanation, err := aiSvc.ExplainDrift(cmd.Context(), report)
		if err != nil {
			return fmt.Errorf("failed to explain drift: %w", err)
		}

		fmt.Println("\n--- Drift Analysis ---")
		fmt.Println(explanation)
		fmt.Println("----------------------")
		return nil
	},
}

var driftDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Check for discrepancies between the current Spec and Plan",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")

		cwd, _ := os.Getwd()
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		service := application.NewDriftService(repo)

		report, err := service.DetectDrift()
		if err != nil {
			return fmt.Errorf("failed to detect drift: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			if len(report.Issues) > 0 {
				return fmt.Errorf("drift detected")
			}
			return nil
		}

		if len(report.Issues) == 0 {
			fmt.Println("No drift detected. Project is in a healthy state.")
			return nil
		}

		fmt.Printf("Detected %d drift issues:\n", len(report.Issues))
		for _, issue := range report.Issues {
			fmt.Printf("- [%s] (%s/%s) %s\n", issue.Severity, issue.Type, issue.Category, issue.Message)
			if issue.Hint != "" {
				fmt.Printf("  Hint: %s\n", issue.Hint)
			}
		}

		return fmt.Errorf("drift detected")
	},
}

var driftAcceptCmd = &cobra.Command{
	Use:   "accept",
	Short: "Accept current drift by locking the spec snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		spec, err := repo.LoadSpec()
		if err != nil {
			return fmt.Errorf("failed to load spec: %w", err)
		}
		if spec == nil {
			return fmt.Errorf("no spec found to lock")
		}

		if err := repo.SaveSpecLock(spec); err != nil {
			return fmt.Errorf("failed to update spec lock: %w", err)
		}

		if err := workspace.Audit.Log("drift.accepted", "cli", map[string]interface{}{
			"spec_id":   spec.ID,
			"spec_hash": spec.Hash(),
		}); err != nil {
			return fmt.Errorf("failed to write audit log: %w", err)
		}

		fmt.Println("Drift accepted. Spec snapshot locked.")
		return nil
	},
}

func init() {
	driftDetectCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	driftCmd.AddCommand(driftDetectCmd)
	driftCmd.AddCommand(driftExplainCmd)
	driftCmd.AddCommand(driftAcceptCmd)
	RootCmd.AddCommand(driftCmd)
}
