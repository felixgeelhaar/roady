package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/ai"
	"github.com/felixgeelhaar/roady/pkg/storage"
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
		repo := storage.NewFilesystemRepository(cwd)
		driftSvc := application.NewDriftService(repo)
		audit := application.NewAuditService(repo)

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
		repo := storage.NewFilesystemRepository(cwd)
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

func init() {
	driftDetectCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	driftCmd.AddCommand(driftDetectCmd)
	driftCmd.AddCommand(driftExplainCmd)
	RootCmd.AddCommand(driftCmd)
}
