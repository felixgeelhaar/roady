package storage

import (
	"errors"
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestSaveState_OptimisticLocking(t *testing.T) {
	dir := t.TempDir()
	repo := NewFilesystemRepository(dir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	// First save: version 0 → becomes 1
	state := planning.NewExecutionState("test")
	if err := repo.SaveState(state); err != nil {
		t.Fatal(err)
	}
	if state.Version != 1 {
		t.Errorf("expected version 1, got %d", state.Version)
	}

	// Reload and save again: version 1 → becomes 2
	loaded, err := repo.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 1 {
		t.Errorf("expected loaded version 1, got %d", loaded.Version)
	}
	if err := repo.SaveState(loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 2 {
		t.Errorf("expected version 2, got %d", loaded.Version)
	}
}

func TestSaveState_ConflictDetected(t *testing.T) {
	dir := t.TempDir()
	repo := NewFilesystemRepository(dir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Create initial state
	state := planning.NewExecutionState("test")
	if err := repo.SaveState(state); err != nil {
		t.Fatal(err)
	}

	// Two readers load the same version
	reader1, err := repo.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	reader2, err := repo.LoadState()
	if err != nil {
		t.Fatal(err)
	}

	// Reader 1 saves successfully
	reader1.SetTaskStatus("task-1", planning.StatusInProgress)
	if err := repo.SaveState(reader1); err != nil {
		t.Fatal(err)
	}

	// Reader 2 has stale version — should conflict
	reader2.SetTaskStatus("task-2", planning.StatusInProgress)
	err = repo.SaveState(reader2)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	var conflictErr *planning.ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected ConflictError, got %T: %v", err, err)
	}
	if conflictErr.Expected != 1 || conflictErr.Actual != 2 {
		t.Errorf("expected conflict(1,2), got conflict(%d,%d)", conflictErr.Expected, conflictErr.Actual)
	}
}

func TestSaveState_NoConflictOnNewFile(t *testing.T) {
	dir := t.TempDir()
	repo := NewFilesystemRepository(dir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Remove state file if it exists
	path, _ := repo.ResolvePath(StateFile)
	_ = os.Remove(path)

	state := planning.NewExecutionState("test")
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("expected no error on first save, got %v", err)
	}
}
