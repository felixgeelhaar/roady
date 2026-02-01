package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [question]",
	Short: "Ask a natural language question about the project",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		if services.AI == nil {
			return MapError(fmt.Errorf("AI service not available; configure an AI provider"))
		}

		question := strings.Join(args, " ")
		answer, err := services.AI.QueryProject(cmd.Context(), question)
		if err != nil {
			return MapError(fmt.Errorf("failed to query project: %w", err))
		}

		fmt.Println(answer)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(queryCmd)
}
