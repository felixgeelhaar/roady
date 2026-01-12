package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git-based automation for Roady",
}

var gitSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Scan recent git commits for task completion markers [roady:task-id]",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		audit := application.NewAuditService(repo)
		taskSvc := application.NewTaskService(repo, audit)

		// 1. Get recent commits (last 10)
		out, err := exec.Command("git", "log", "-n", "10", "--pretty=format:%H|%s").Output()
		if err != nil {
			return fmt.Errorf("failed to read git log: %w (are you in a git repo?)", err)
		}

		lines := strings.Split(string(out), "\n")
		fmt.Printf("Scanning %d recent commits for Roady markers...\n", len(lines))

		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) < 2 { continue }
			hash, message := parts[0], parts[1]

			// Look for [roady:task-id]
			if strings.Contains(message, "[roady:") {
				start := strings.Index(message, "[roady:") + 7
				end := strings.Index(message[start:], "]")
				if end != -1 {
					taskID := message[start : start+end]
					fmt.Printf("Found marker for task '%s' in commit %s\n", taskID, hash[:8])
					
					// Transition to complete
				err := taskSvc.TransitionTask(taskID, "complete", "git-automation", "Commit: "+hash)
				if err != nil {
						// It might already be complete, or blocked
						fmt.Printf("  Could not transition: %v\n", err)
					} else {
						fmt.Printf("  Task %s marked as complete via git.\n", taskID)
					}
				}
			}
		}

		return nil
	},
}

func init() {
	gitCmd.AddCommand(gitSyncCmd)
	RootCmd.AddCommand(gitCmd)
}