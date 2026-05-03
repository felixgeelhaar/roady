package cli

import (
	"context"
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
		var answer string
		err = withAIProgress(cmd.Context(), "AI query", func(ctx context.Context) error {
			a, qerr := services.AI.QueryProject(ctx, question)
			answer = a
			return qerr
		})
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
