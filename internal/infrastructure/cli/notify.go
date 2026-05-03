package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// notifyCmd is the canonical home for outbound notification configuration.
// It supersedes both `roady messaging` and `roady webhook notif`, which
// remain registered as deprecation aliases that delegate to the same
// handlers in notify_ops.go.
var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Configure outbound notifications (webhook, Slack, etc.)",
	Long: `Manage outbound notification adapters.

Supported types: webhook, slack.

The legacy commands ` + "`roady messaging`" + ` and ` + "`roady webhook notif`" + ` are
preserved as deprecation aliases that delegate to the same implementation.`,
}

var notifyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured notification adapters",
	RunE: func(cmd *cobra.Command, args []string) error {
		return notifyList(os.Stdout)
	},
}

var notifyAddCmd = &cobra.Command{
	Use:   "add <name> <type> <url>",
	Short: "Add a notification adapter (types: webhook, slack)",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notifyAdd(os.Stdout, args[0], args[1], args[2])
	},
}

var notifyTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Send a test event through a notification adapter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notifyTest(os.Stdout, args[0])
	},
}

var notifyRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a notification adapter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notifyRemove(os.Stdout, args[0])
	},
}

func init() {
	notifyCmd.AddCommand(notifyListCmd)
	notifyCmd.AddCommand(notifyAddCmd)
	notifyCmd.AddCommand(notifyTestCmd)
	notifyCmd.AddCommand(notifyRemoveCmd)
	RootCmd.AddCommand(notifyCmd)
}
