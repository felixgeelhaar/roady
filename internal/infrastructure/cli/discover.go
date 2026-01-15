package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover [root-dir]",
	Short: "Discover all Roady projects in a directory tree",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		fmt.Printf("Searching for Roady projects in: %s\n", root)

		projects := []string{}
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() && info.Name() == ".roady" {
				projects = append(projects, filepath.Dir(path))
				return filepath.SkipDir // Don't look inside .roady
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No Roady projects found.")
			return nil
		}

		fmt.Printf("Found %d Roady projects:\n", len(projects))
		for _, p := range projects {
			abs, _ := filepath.Abs(p)
			fmt.Printf("- %s\n", abs)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(discoverCmd)
}
