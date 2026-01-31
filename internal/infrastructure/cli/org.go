package cli

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var orgJSON bool

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

		svc := application.NewOrgService(root)
		metrics, err := svc.AggregateMetrics()
		if err != nil {
			return err
		}

		if len(metrics.Projects) == 0 {
			fmt.Println("No Roady projects found.")
			return nil
		}

		if orgJSON {
			data, err := json.MarshalIndent(metrics, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
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
		for _, pm := range metrics.Projects {
			progress := "0%"
			if pm.Total > 0 {
				progress = fmt.Sprintf("%.1f%%", pm.Progress)
			}
			rows = append(rows, table.Row{
				pm.Name,
				progress,
				fmt.Sprintf("%d", pm.Verified),
				fmt.Sprintf("%d", pm.WIP),
				fmt.Sprintf("%d", pm.Total),
				pm.Path,
			})
		}

		// Summary row
		avgProgress := "0%"
		if metrics.TotalProjects > 0 {
			avgProgress = fmt.Sprintf("%.1f%%", metrics.AvgProgress)
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("TOTAL (%d)", metrics.TotalProjects),
			avgProgress,
			fmt.Sprintf("%d", metrics.TotalVerified),
			fmt.Sprintf("%d", metrics.TotalWIP),
			fmt.Sprintf("%d", metrics.TotalTasks),
			"",
		})

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithHeight(len(rows)+1),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Bold(true)
		s.Selected = lipgloss.NewStyle()
		t.SetStyles(s)

		fmt.Printf("Organizational Status (%d projects)\n", metrics.TotalProjects)
		fmt.Println(t.View())
		return nil
	},
}

func init() {
	orgStatusCmd.Flags().BoolVar(&orgJSON, "json", false, "Output as JSON")
	orgCmd.AddCommand(orgStatusCmd)
	RootCmd.AddCommand(orgCmd)
}
