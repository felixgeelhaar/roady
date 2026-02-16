package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version     = "dev"
	Commit      = "none"
	Date        = "unknown"
	projectPath string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:           "roady",
	Version:       Version,
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "A planning-first system of record for software work",
	Long: `Roady is a planning-first system of record for software work.
It helps individuals and teams answer:
1. What are we building?
2. Why are we building it?
3. What should happen next?`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() error {
	err := RootCmd.Execute()
	if err == nil {
		return nil
	}

	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		fmt.Fprintf(os.Stderr, "Error: %s\n", cliErr.Message)
		if cliErr.Hint != "" {
			fmt.Fprintf(os.Stderr, "Hint:  %s\n", cliErr.Hint)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return err
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&projectPath, "path", "C", "",
		"Path to the roady project directory (defaults to current directory)")
}
