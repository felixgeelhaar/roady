package cli

import (
	"os"
	"os/exec"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestGitSyncCmd_FindsMarkers(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("git config email: %v", err)
	}
	if err := exec.Command("git", "config", "user.name", "Tester").Run(); err != nil {
		t.Fatalf("git config name: %v", err)
	}
	if err := exec.Command("git", "config", "commit.gpgsign", "false").Run(); err != nil {
		t.Fatalf("git config gpgsign: %v", err)
	}

	if err := os.WriteFile("README.md", []byte("test"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := exec.Command("git", "add", "README.md").Run(); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := exec.Command("git", "commit", "-m", "Update docs [roady:task-123]").Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	if err := repo.SavePlan(&planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "task-123", Title: "Task"},
		},
	}); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	state := planning.NewExecutionState("p1")
	state.TaskStates["task-123"] = planning.TaskResult{Status: planning.StatusInProgress}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := gitSyncCmd.RunE(gitSyncCmd, []string{}); err != nil {
		t.Fatalf("git sync failed: %v", err)
	}

	state, err := repo.LoadState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.TaskStates["task-123"].Status != planning.StatusDone {
		t.Fatalf("expected task to be done, got %s", state.TaskStates["task-123"].Status)
	}
}
