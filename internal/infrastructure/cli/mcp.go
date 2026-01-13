package cli

import (
	"fmt"
	"os"
	"strings"

	inframcp "github.com/felixgeelhaar/roady/internal/infrastructure/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpTransport string
	mcpAddr      string
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Roady MCP server",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("ROADY_SKIP_MCP_START") == "true" {
			return
		}
		cwd, _ := os.Getwd()
		server := inframcp.NewServer(cwd)
		var err error
		switch strings.ToLower(mcpTransport) {
		case "stdio", "":
			err = server.StartStdio()
		case "http":
			err = server.StartHTTP(mcpAddr)
		case "ws", "websocket":
			err = server.StartWebSocket(mcpAddr)
		default:
			err = fmt.Errorf("unsupported transport: %s", mcpTransport)
		}
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	mcpCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "Transport to use (stdio, http, ws)")
	mcpCmd.Flags().StringVar(&mcpAddr, "addr", ":8080", "Address for http/ws transports")
	RootCmd.AddCommand(mcpCmd)
}
