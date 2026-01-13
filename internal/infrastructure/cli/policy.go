package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage and enforce policies",
}

var policyCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if the current plan complies with policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repo := wiring.NewWorkspace(cwd).Repo
		service := application.NewPolicyService(repo)

		violations, err := service.CheckCompliance()
		if err != nil {
			return fmt.Errorf("failed to check policy: %w", err)
		}

		if len(violations) == 0 {
			fmt.Println("No policy violations found.")
			return nil
		}

		fmt.Printf("Found %d policy violations:\n", len(violations))
		for _, v := range violations {
			color := ""
			if v.Level == policy.ViolationError {
				color = "[ERROR]"
			} else {
				color = "[WARN]"
			}
			fmt.Printf("%s %s: %s\n", color, v.RuleID, v.Message)
		}

		return fmt.Errorf("policy violations found")
	},
}

func init() {
	policyCmd.AddCommand(policyCheckCmd)
	RootCmd.AddCommand(policyCmd)
}
