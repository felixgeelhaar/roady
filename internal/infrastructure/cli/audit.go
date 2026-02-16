package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit and verify project history",
}

var auditVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify the integrity of the project audit trail",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}
		workspace := wiring.NewWorkspace(cwd)
		service := application.NewAuditService(workspace.Repo)

		fmt.Println("Verifying audit trail integrity...")
		violations, err := service.VerifyIntegrity()
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		if len(violations) == 0 {
			fmt.Println("Audit trail is intact and verified.")
			return nil
		}

		fmt.Printf("Found %d integrity violations:\n", len(violations))
		for _, v := range violations {
			fmt.Printf("  - %s\n", v)
		}
		os.Exit(1)
		return nil
	},
}

func init() {
	auditCmd.AddCommand(auditVerifyCmd)
	RootCmd.AddCommand(auditCmd)
}
