package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Organizational views across multiple projects",
}

var orgStatusCmd = &cobra.Command{
	Use:   "status [root-dir]",
	Short: "Show status overview of all Roady projects in a directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		projects := []string{}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() && info.Name() == ".roady" {
				projects = append(projects, filepath.Dir(path))
				return filepath.SkipDir
			}
			return nil
		})

		if len(projects) == 0 {
			fmt.Println("No Roady projects found.")
			return nil
		}

		columns := []table.Column{
			{Title: "Project", Width: 30},
			{Title: "Progress", Width: 10},
			{Title: "Vrf", Width: 5},
			{Title: "WIP", Width: 5},
			{Title: "Total", Width: 5},
			{Title: "Path", Width: 40},
		}

	
rows := []table.Row{}
		for _, p := range projects {
			repo := storage.NewFilesystemRepository(p)
			spec, _ := repo.LoadSpec()
			plan, _ := repo.LoadPlan()
			state, _ := repo.LoadState()

			name := filepath.Base(p)
			if spec != nil {
				name = spec.Title
			}

			verified, wip, total := 0, 0, 0
			progress := "0%"
			if plan != nil {
				total = len(plan.Tasks)
				for _, t := range plan.Tasks {
					if state != nil {
						if res, ok := state.TaskStates[t.ID]; ok {
							if res.Status == planning.StatusVerified {
								verified++
							}
							if res.Status == planning.StatusInProgress {
								wip++
							}
						}
					}
				}
				if total > 0 {
					progress = fmt.Sprintf("%.1f%%", float64(verified)/float64(total)*100)
				}
			}

			abs, _ := filepath.Abs(p)
			rows = append(rows, table.Row{
				name,
				progress,
				fmt.Sprintf("%d", verified),
				fmt.Sprintf("%d", wip),
				fmt.Sprintf("%d", total),
				abs,
			})
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithHeight(len(rows) + 1),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Bold(true)
		s.Selected = lipgloss.NewStyle() // Disable selection style for static view
		t.SetStyles(s)

		fmt.Printf("🏢 Organizational Status (%d projects)\n", len(projects))
		fmt.Println(t.View())
		return nil
	},
}

func init() {
	orgCmd.AddCommand(orgStatusCmd)
	RootCmd.AddCommand(orgCmd)
}
