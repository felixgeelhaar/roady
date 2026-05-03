package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// `roady messaging` is retained as a deprecation alias for `roady notify`.
// Both surfaces share the implementations in notify_ops.go; the only thing
// preserved here is the legacy command path so existing scripts keep
// working. A one-line migration hint is printed before each invocation.

const messagingDeprecationHint = "Note: `roady messaging` is deprecated; use `roady notify` instead."

func messagingDeprecation(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, messagingDeprecationHint)
}

var messagingCmd = &cobra.Command{
	Use:              "messaging",
	Short:            "DEPRECATED: use `roady notify`. Manage messaging adapters.",
	PersistentPreRun: messagingDeprecation,
}

var messagingListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured messaging adapters",
	RunE: func(cmd *cobra.Command, args []string) error {
		return notifyList(os.Stdout)
	},
}

var messagingAddCmd = &cobra.Command{
	Use:   "add <name> <type> <url>",
	Short: "Add a messaging adapter (types: webhook, slack)",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notifyAdd(os.Stdout, args[0], args[1], args[2])
	},
}

var messagingTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Send a test event to a messaging adapter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notifyTest(os.Stdout, args[0])
	},
}

func init() {
	messagingCmd.AddCommand(messagingListCmd)
	messagingCmd.AddCommand(messagingAddCmd)
	messagingCmd.AddCommand(messagingTestCmd)
	RootCmd.AddCommand(messagingCmd)
}
