package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/storage"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage individual tasks",
}

func createTaskCommand(use, short, event string) *cobra.Command {
	var evidence string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			repo := storage.NewFilesystemRepository(cwd)
			audit := application.NewAuditService(repo)
			service := application.NewTaskService(repo, audit)
			taskID := args[0]

			actor := os.Getenv("USER")
			if actor == "" {
				actor = "unknown-human"
			}

			err := service.TransitionTask(taskID, event, actor, evidence)
			if err != nil {
				return fmt.Errorf("failed to transition task: %w", err)
			}
			fmt.Printf("Task %s transition '%s' successful.\n", taskID, event)
			return nil
		},
	}
	cmd.Flags().StringVarP(&evidence, "evidence", "e", "", "Evidence for the task completion (e.g. commit hash, URL)")
	return cmd
}

func init() {
	taskCmd.AddCommand(createTaskCommand("start", "Start a task", "start"))
	taskCmd.AddCommand(createTaskCommand("block", "Block a task", "block"))
	taskCmd.AddCommand(createTaskCommand("unblock", "Unblock a task", "unblock"))
	taskCmd.AddCommand(createTaskCommand("complete", "Complete a task", "complete"))
	taskCmd.AddCommand(createTaskCommand("stop", "Stop working on a task", "stop"))
	taskCmd.AddCommand(createTaskCommand("reopen", "Reopen a completed task", "reopen"))
	taskCmd.AddCommand(createTaskCommand("verify", "Mark a task as verified with evidence", "verify"))

	RootCmd.AddCommand(taskCmd)
}
