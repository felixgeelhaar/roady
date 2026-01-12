package cli

import (
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "roady",
	Version: Version,
	Short:   "A planning-first system of record for software work",
	Long: `Roady is a planning-first system of record for software work.
It helps individuals and teams answer:
1. What are we building?
2. Why are we building it?
3. What should happen next?`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	// Global flags can be defined here
}
