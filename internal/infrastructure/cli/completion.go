package cli

import (
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generate shell completion scripts",
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenZshCompletion(cmd.OutOrStdout())
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
	},
}

var completionPowershellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate powershell completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
	},
}

func init() {
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	completionCmd.AddCommand(completionPowershellCmd)
	RootCmd.AddCommand(completionCmd)
}
