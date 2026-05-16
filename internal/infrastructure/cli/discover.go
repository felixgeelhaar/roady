package cli

import (
	"fmt"
	"path/filepath"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover [root-dir]",
	Short: "Discover all Roady projects in a directory tree",
	Long: `Discover all Roady projects (and named sub-projects) in a directory tree.

Each ".roady/" directory found surfaces its root project, plus every
named sub-project under ".roady/projects/<name>/".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		fmt.Printf("Searching for Roady projects in: %s\n", root)

		svc := application.NewOrgService(root)
		projects, err := svc.DiscoverProjectsWithSub()
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No Roady projects found.")
			return nil
		}

		fmt.Printf("Found %d Roady projects:\n", len(projects))
		for _, p := range projects {
			abs, _ := filepath.Abs(p.Path)
			if p.SubProject == "" {
				fmt.Printf("- %s\n", abs)
				continue
			}
			fmt.Printf("- %s  (sub-project: %s)\n",
				filepath.Join(abs, ".roady", "projects", p.SubProject),
				p.SubProject,
			)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(discoverCmd)
}
