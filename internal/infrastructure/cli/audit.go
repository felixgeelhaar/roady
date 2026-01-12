package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
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
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		service := application.NewAuditService(repo)

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