package dependency

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewResolver(t *testing.T) {
	r := NewResolver("/tmp/test-repo")
	if r.rootPath != "/tmp/test-repo" {
		t.Errorf("rootPath = %s, want /tmp/test-repo", r.rootPath)
	}
}

func TestResolver_ParseSpecDependencies(t *testing.T) {
	r := NewResolver("/tmp/root")

	specDeps := []SpecDependency{
		{Repo: "../service-a", Type: "runtime", Description: "API dependency"},
		{Repo: "../service-b", Type: "data", Features: []string{"feat-1"}},
	}

	deps, errs := r.ParseSpecDependencies(specDeps)
	if len(errs) != 0 {
		t.Errorf("Unexpected errors: %v", errs)
	}
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	// Check first dependency
	if deps[0].Type != DependencyRuntime {
		t.Errorf("First dep type = %s, want runtime", deps[0].Type)
	}
	if deps[0].Description != "API dependency" {
		t.Errorf("First dep description = %s, want 'API dependency'", deps[0].Description)
	}

	// Check second dependency
	if deps[1].Type != DependencyData {
		t.Errorf("Second dep type = %s, want data", deps[1].Type)
	}
	if len(deps[1].FeatureIDs) != 1 || deps[1].FeatureIDs[0] != "feat-1" {
		t.Errorf("Second dep features = %v, want [feat-1]", deps[1].FeatureIDs)
	}
}

func TestResolver_ParseSpecDependencies_InvalidType(t *testing.T) {
	r := NewResolver("/tmp/root")

	specDeps := []SpecDependency{
		{Repo: "../service", Type: "invalid-type"},
	}

	deps, errs := r.ParseSpecDependencies(specDeps)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errs))
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies on error, got %d", len(deps))
	}
}

func TestResolver_ParseSpecDependencies_EmptyRepo(t *testing.T) {
	r := NewResolver("/tmp/root")

	specDeps := []SpecDependency{
		{Repo: "", Type: "runtime"},
	}

	deps, errs := r.ParseSpecDependencies(specDeps)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for empty repo, got %d", len(errs))
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies on error, got %d", len(deps))
	}
}

func TestResolver_ParseSpecDependencies_SelfDependency(t *testing.T) {
	r := NewResolver("/tmp/root")

	specDeps := []SpecDependency{
		{Repo: ".", Type: "runtime"},
	}

	deps, errs := r.ParseSpecDependencies(specDeps)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for self-dependency, got %d", len(errs))
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies on error, got %d", len(deps))
	}
}

func TestResolver_resolveRepoPath_Relative(t *testing.T) {
	r := NewResolver("/home/user/project")

	tests := []struct {
		input    string
		expected string
	}{
		{"../other", "/home/user/other"},
		{"./subdir", "/home/user/project/subdir"},
		{"../../shared/lib", "/home/shared/lib"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := r.resolveRepoPath(tt.input)
			if got != tt.expected {
				t.Errorf("resolveRepoPath(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolver_resolveRepoPath_Absolute(t *testing.T) {
	r := NewResolver("/home/user/project")

	input := "/absolute/path/to/repo"
	got := r.resolveRepoPath(input)
	if got != input {
		t.Errorf("resolveRepoPath(%s) = %s, want %s", input, got, input)
	}
}

func TestResolver_isSameRepo(t *testing.T) {
	// Use a temp directory that actually exists for accurate filepath.Abs behavior
	tmpDir := t.TempDir()
	r := NewResolver(tmpDir)

	// Create a sibling directory
	siblingDir := filepath.Join(filepath.Dir(tmpDir), "sibling")
	os.MkdirAll(siblingDir, 0755)

	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		{"same absolute path", tmpDir, true},
		{"same path with trailing slash", tmpDir + "/", true},
		{"different path", siblingDir, false},
		{"resolved current dir", r.resolveRepoPath("."), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.isSameRepo(tt.target)
			if got != tt.expected {
				t.Errorf("isSameRepo(%s) = %v, want %v", tt.target, got, tt.expected)
			}
		})
	}
}

func TestResolver_ValidateDependencyPath(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	depDir := filepath.Join(tmpDir, "dep-repo")
	os.MkdirAll(depDir, 0755)

	r := NewResolver(tmpDir)

	// Valid directory
	err := r.ValidateDependencyPath(depDir)
	if err != nil {
		t.Errorf("Expected nil error for valid path, got %v", err)
	}

	// Non-existent directory
	err = r.ValidateDependencyPath(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestResolver_ValidateDependencyPath_NotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "somefile.txt")
	os.WriteFile(filePath, []byte("content"), 0644)

	r := NewResolver(tmpDir)

	err := r.ValidateDependencyPath(filePath)
	if err == nil {
		t.Error("Expected error for file path (not directory)")
	}
}

func TestResolver_ResolveDependencyGraph(t *testing.T) {
	r := NewResolver("/tmp/root")

	specDeps := []SpecDependency{
		{Repo: "../service-a", Type: "runtime"},
		{Repo: "../service-b", Type: "data"},
	}

	graph, errs := r.ResolveDependencyGraph(specDeps)
	if len(errs) != 0 {
		t.Errorf("Unexpected errors: %v", errs)
	}
	if graph.RootRepo != "/tmp/root" {
		t.Errorf("RootRepo = %s, want /tmp/root", graph.RootRepo)
	}
	if len(graph.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(graph.Dependencies))
	}
}

func TestResolver_DiscoverRoadyRepos(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create some roady repos
	repo1 := filepath.Join(tmpDir, "repo1", ".roady")
	repo2 := filepath.Join(tmpDir, "sub", "repo2", ".roady")
	os.MkdirAll(repo1, 0755)
	os.MkdirAll(repo2, 0755)

	// Create a non-roady dir
	os.MkdirAll(filepath.Join(tmpDir, "not-roady"), 0755)

	r := NewResolver(tmpDir)

	repos, err := r.DiscoverRoadyRepos(tmpDir, 3)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("Expected 2 roady repos, got %d: %v", len(repos), repos)
	}
}

func TestResolver_ValidateDependenciesWithDetails(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid repo
	validRepo := filepath.Join(tmpDir, "valid-repo")
	os.MkdirAll(validRepo, 0755)

	// Create roady repo
	roadyRepo := filepath.Join(tmpDir, "roady-repo")
	os.MkdirAll(filepath.Join(roadyRepo, ".roady"), 0755)

	r := NewResolver(tmpDir)

	deps := []*RepoDependency{
		NewRepoDependency(tmpDir, validRepo, DependencyRuntime),
		NewRepoDependency(tmpDir, roadyRepo, DependencyRuntime),
		NewRepoDependency(tmpDir, filepath.Join(tmpDir, "missing"), DependencyRuntime),
	}

	results := r.ValidateDependenciesWithDetails(deps)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// First should be valid, not roady
	if !results[0].IsValid {
		t.Error("First dependency should be valid")
	}
	if results[0].IsRoadyRepo {
		t.Error("First dependency should not be a roady repo")
	}

	// Second should be valid and roady
	if !results[1].IsValid {
		t.Error("Second dependency should be valid")
	}
	if !results[1].IsRoadyRepo {
		t.Error("Second dependency should be a roady repo")
	}

	// Third should be invalid
	if results[2].IsValid {
		t.Error("Third dependency should be invalid")
	}
	if results[2].Error == nil {
		t.Error("Third dependency should have an error")
	}
}

func TestResolver_ValidateAllDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	validRepo := filepath.Join(tmpDir, "valid")
	os.MkdirAll(validRepo, 0755)

	r := NewResolver(tmpDir)

	deps := []*RepoDependency{
		NewRepoDependency(tmpDir, validRepo, DependencyRuntime),
		NewRepoDependency(tmpDir, filepath.Join(tmpDir, "missing"), DependencyRuntime),
	}

	errors := r.ValidateAllDependencies(deps)

	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
}
