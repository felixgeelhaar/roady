package cli

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var timelineCmd = &cobra.Command{
	Use:   "timeline",
	Short: "Show a chronological view of project activity",
	RunE:  runTimeline,
}

func runTimeline(cmd *cobra.Command, args []string) error {
	cwd, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("resolve project path: %w", err)
	}
	workspace := wiring.NewWorkspace(cwd)
	service := application.NewAuditService(workspace.Repo)

	events, err := service.GetTimeline()
	if err != nil {
		return fmt.Errorf("failed to load timeline: %w", err)
	}

	fmt.Println("Project Timeline")
	fmt.Println("------------------")
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		ts := e.Timestamp.Format(time.RFC822)
		fmt.Printf("[%s] %-15s | %-15s", ts, e.Actor, e.Action)
		if len(e.Metadata) > 0 {
			fmt.Printf(" (%v)", e.Metadata)
		}
		fmt.Println()
	}
	return nil
}

// RunTimeline is the exported RunE function for use as a subcommand
var RunTimeline = runTimeline

func init() {
	timelineCmd.Hidden = true // Hide from top-level help, available via `status timeline`
	RootCmd.AddCommand(timelineCmd)
}
