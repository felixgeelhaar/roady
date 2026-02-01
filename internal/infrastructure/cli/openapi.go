package cli

import (
	"fmt"
	"os"

	mcpserver "github.com/felixgeelhaar/roady/internal/infrastructure/mcp"
	"github.com/spf13/cobra"
)

var openapiCmd = &cobra.Command{
	Use:   "openapi",
	Short: "Generate an OpenAPI 3.0 spec from MCP tool registrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		srv, err := mcpserver.NewServer(cwd)
		if err != nil {
			return MapError(fmt.Errorf("failed to initialize server: %w", err))
		}

		data, err := srv.OpenAPI()
		if err != nil {
			return MapError(fmt.Errorf("failed to generate OpenAPI spec: %w", err))
		}

		fmt.Println(string(data))
		return nil
	},
}

func init() {
	RootCmd.AddCommand(openapiCmd)
}
