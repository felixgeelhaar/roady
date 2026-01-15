package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var autoSync bool

var watchCmd = &cobra.Command{

	Use: "watch [dir]",

	Short: "Watch a directory for documentation changes and automatically detect drift",

	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "docs"

		if len(args) > 0 {
			dir = args[0]
		}

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		specSvc := services.Spec
		driftSvc := services.Drift
		aiSvc := services.AI

		fmt.Printf("Watching %s for changes... (Auto-sync: %v)\n", dir, autoSync)

		lastHash := ""
		if seed := os.Getenv("ROADY_WATCH_SEED_HASH"); seed != "" {
			lastHash = seed
		}
		once := os.Getenv("ROADY_WATCH_ONCE") == "true"
		for {

			currentSpec, err := specSvc.AnalyzeDirectory(dir)
			if err == nil {

				currentHash := currentSpec.Hash()
				if currentHash != lastHash {

					if lastHash != "" {
						fmt.Printf("\nDocumentation change detected at %s\n", time.Now().Format("15:04:05"))

						if autoSync {
							fmt.Println("Autonomous Reconciliation: Synchronizing plan with new intent...")

							if _, err := aiSvc.DecomposeSpec(cmd.Context()); err != nil {
								fmt.Printf("Auto-sync failed: %v\n", err)
							} else {
								fmt.Println("Plan successfully synchronized.")
							}
						}

						// 2. Detect Drift Automatically
						report, err := driftSvc.DetectDrift(cmd.Context())
						if err == nil && len(report.Issues) > 0 {
							fmt.Printf("Drift detected: %d issues found.\n", len(report.Issues))
							for _, iss := range report.Issues {
								fmt.Printf("  - [%s] %s\n", iss.Severity, iss.Message)
							}
						} else {
							fmt.Println("Intent and Plan are in sync.")
						}

					}
					lastHash = currentHash
				}

			}

			time.Sleep(2 * time.Second)
			if once {
				return nil
			}

		}

	},
}

func init() {
	watchCmd.Flags().BoolVar(&autoSync, "auto-sync", false, "Automatically regenerate plan on documentation changes")
	RootCmd.AddCommand(watchCmd)
}
