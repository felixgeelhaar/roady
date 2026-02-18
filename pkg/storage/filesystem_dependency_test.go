package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/dependency"
)

func TestFilesystemRepository_SaveLoadDependencyGraph(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	graph := dependency.NewDependencyGraph(tmpDir)
	graph.AddDependency(dependency.NewRepoDependency(tmpDir, "/other/repo", dependency.DependencyRuntime))

	// Save
	err := repo.SaveDependencyGraph(graph)
	if err != nil {
		t.Fatalf("SaveDependencyGraph failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, RoadyDir, DependenciesFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Dependencies file was not created")
	}

	// Load
	loaded, err := repo.LoadDependencyGraph()
	if err != nil {
		t.Fatalf("LoadDependencyGraph failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Loaded graph is nil")
	}
	if len(loaded.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(loaded.Dependencies))
	}
}

func TestFilesystemRepository_LoadDependencyGraph_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	graph, err := repo.LoadDependencyGraph()
	if err != nil {
		t.Fatalf("LoadDependencyGraph failed: %v", err)
	}
	if graph != nil {
		t.Error("Expected nil graph when file doesn't exist")
	}
}

func TestFilesystemRepository_AddDependency(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	dep := dependency.NewRepoDependency(tmpDir, "/other/repo", dependency.DependencyRuntime)

	// Add first dependency
	err := repo.AddDependency(dep)
	if err != nil {
		t.Fatalf("AddDependency failed: %v", err)
	}

	// Verify it was saved
	graph, _ := repo.LoadDependencyGraph()
	if len(graph.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(graph.Dependencies))
	}

	// Add second dependency
	dep2 := dependency.NewRepoDependency(tmpDir, "/another/repo", dependency.DependencyData)
	err = repo.AddDependency(dep2)
	if err != nil {
		t.Fatalf("AddDependency (second) failed: %v", err)
	}

	// Verify both exist
	graph, _ = repo.LoadDependencyGraph()
	if len(graph.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(graph.Dependencies))
	}
}

func TestFilesystemRepository_RemoveDependency(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	dep := dependency.NewRepoDependency(tmpDir, "/other/repo", dependency.DependencyRuntime)
	if err := repo.AddDependency(dep); err != nil {
		t.Fatal(err)
	}

	// Remove existing
	err := repo.RemoveDependency(dep.ID)
	if err != nil {
		t.Fatalf("RemoveDependency failed: %v", err)
	}

	graph, _ := repo.LoadDependencyGraph()
	if len(graph.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies after removal, got %d", len(graph.Dependencies))
	}
}

func TestFilesystemRepository_RemoveDependency_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Remove from empty graph
	err := repo.RemoveDependency("nonexistent")
	if err != dependency.ErrDependencyNotFound {
		t.Errorf("Expected ErrDependencyNotFound, got %v", err)
	}
}

func TestFilesystemRepository_GetDependency(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	dep := dependency.NewRepoDependency(tmpDir, "/other/repo", dependency.DependencyRuntime)
	if err := repo.AddDependency(dep); err != nil {
		t.Fatal(err)
	}

	// Get existing
	found, err := repo.GetDependency(dep.ID)
	if err != nil {
		t.Fatalf("GetDependency failed: %v", err)
	}
	if found == nil {
		t.Fatal("Expected dependency to be found")
	}
	if found.ID != dep.ID {
		t.Errorf("ID mismatch: got %s, want %s", found.ID, dep.ID)
	}

	// Get non-existent
	notFound, err := repo.GetDependency("nonexistent")
	if err != nil {
		t.Fatalf("GetDependency (nonexistent) failed: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent dependency")
	}
}

func TestFilesystemRepository_ListDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}

	// List empty
	deps, err := repo.ListDependencies()
	if err != nil {
		t.Fatalf("ListDependencies failed: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}

	// Add and list
	if err := repo.AddDependency(dependency.NewRepoDependency(tmpDir, "/a", dependency.DependencyRuntime)); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddDependency(dependency.NewRepoDependency(tmpDir, "/b", dependency.DependencyData)); err != nil {
		t.Fatal(err)
	}

	deps, err = repo.ListDependencies()
	if err != nil {
		t.Fatalf("ListDependencies (after add) failed: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}
}

func TestFilesystemRepository_UpdateRepoHealth(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil { t.Fatal(err) }

	health := &dependency.RepoHealth{
		RepoPath:       "/some/repo",
		IsReachable:    true,
		HasDrift:       false,
		CompletionRate: 75.0,
	}

	err := repo.UpdateRepoHealth(health)
	if err != nil {
		t.Fatalf("UpdateRepoHealth failed: %v", err)
	}

	// Verify it was saved
	loaded, err := repo.GetRepoHealth("/some/repo")
	if err != nil {
		t.Fatalf("GetRepoHealth failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected health to be found")
	}
	if loaded.CompletionRate != 75.0 {
		t.Errorf("CompletionRate = %f, want 75.0", loaded.CompletionRate)
	}
}

func TestFilesystemRepository_GetRepoHealth_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil { t.Fatal(err) }

	health, err := repo.GetRepoHealth("/nonexistent")
	if err != nil {
		t.Fatalf("GetRepoHealth failed: %v", err)
	}
	if health != nil {
		t.Error("Expected nil health for non-existent repo")
	}
}
