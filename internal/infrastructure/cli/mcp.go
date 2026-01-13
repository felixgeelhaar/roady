package cli

import (
	"os"

	"github.com/felixgeelhaar/roady/pkg/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Roady MCP server",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("ROADY_SKIP_MCP_START") == "true" {
			return
		}
		cwd, _ := os.Getwd()
		server := mcp.NewServer(cwd)
		if err := server.Start(); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
}
