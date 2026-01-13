package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/storage"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new roady project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		audit := application.NewAuditService(repo)
		service := application.NewInitService(repo, audit)

		projectName := "new-project"
		if len(args) > 0 {
			projectName = args[0]
		}

		err := service.InitializeProject(projectName)
		if err != nil {
			return fmt.Errorf("failed to initialize project: %w", err)
		}

		fmt.Printf("Successfully initialized roady project: %s\n", projectName)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
}
