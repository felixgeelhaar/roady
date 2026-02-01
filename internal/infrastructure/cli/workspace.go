package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage shared workspace synchronization",
}

var workspaceJSONOutput bool

var workspacePushCmd = &cobra.Command{
	Use:   "push",
	Short: "Commit and push .roady/ changes to the remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		workspace := wiring.NewWorkspace(cwd)
		svc := application.NewWorkspaceSyncService(cwd, workspace.Audit)

		result, err := svc.Push(cmd.Context())
		if err != nil {
			return MapError(fmt.Errorf("workspace push: %w", err))
		}

		if workspaceJSONOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Println(result.Message)
		for _, f := range result.Files {
			fmt.Printf("  %s\n", f)
		}
		return nil
	},
}

var workspacePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull remote .roady/ changes and merge",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		workspace := wiring.NewWorkspace(cwd)
		svc := application.NewWorkspaceSyncService(cwd, workspace.Audit)

		result, err := svc.Pull(cmd.Context())
		if err != nil {
			return MapError(fmt.Errorf("workspace pull: %w", err))
		}

		if workspaceJSONOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Println(result.Message)
		if result.Conflict {
			fmt.Println("Conflicting files:")
			for _, f := range result.Files {
				fmt.Printf("  %s\n", f)
			}
		}
		return nil
	},
}

func init() {
	workspacePushCmd.Flags().BoolVar(&workspaceJSONOutput, "json", false, "Output in JSON format")
	workspacePullCmd.Flags().BoolVar(&workspaceJSONOutput, "json", false, "Output in JSON format")
	workspaceCmd.AddCommand(workspacePushCmd)
	workspaceCmd.AddCommand(workspacePullCmd)
	RootCmd.AddCommand(workspaceCmd)
}
