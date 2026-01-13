package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new roady project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		audit := workspace.Audit
		service := application.NewInitService(repo, audit)

		projectName := "new-project"
		if len(args) > 0 {
			projectName = args[0]
		}

		err := service.InitializeProject(projectName)
		if err != nil {
			return fmt.Errorf("failed to initialize project: %w", err)
		}

		if initInteractive {
			if err := runAIConfigureInteractive(repo); err != nil {
				return fmt.Errorf("failed to configure AI: %w", err)
			}
		}

		fmt.Printf("Successfully initialized roady project: %s\n", projectName)
		return nil
	},
}

var initInteractive bool

func init() {
	initCmd.Flags().BoolVar(&initInteractive, "interactive", false, "Prompt for AI configuration after initialization")
	RootCmd.AddCommand(initCmd)
}
