package dependency

import (
	"fmt"
	"os"
	"path/filepath"
)

// SpecDependency represents a dependency declared in a spec file.
type SpecDependency struct {
	// Repo is the path to the dependent repository (relative or absolute).
	Repo string `json:"repo" yaml:"repo"`
	// Type is the dependency type (runtime, data, build, intent).
	Type string `json:"type" yaml:"type"`
	// Description explains why this dependency exists.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Features lists feature IDs that establish this dependency.
	Features []string `json:"features,omitempty" yaml:"features,omitempty"`
}

// SpecDependencies holds dependencies declared in a spec file.
type SpecDependencies struct {
	Dependencies []SpecDependency `json:"dependencies" yaml:"dependencies"`
}

// Resolver parses and resolves dependencies from spec configurations.
type Resolver struct {
	// rootPath is the root path of the current repository.
	rootPath string
}

// NewResolver creates a new dependency resolver.
func NewResolver(rootPath string) *Resolver {
	return &Resolver{
		rootPath: rootPath,
	}
}

// ParseSpecDependencies converts spec dependencies to domain RepoDependencies.
func (r *Resolver) ParseSpecDependencies(specDeps []SpecDependency) ([]*RepoDependency, []error) {
	var dependencies []*RepoDependency
	var errors []error

	for _, specDep := range specDeps {
		dep, err := r.parseSpecDependency(specDep)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		dependencies = append(dependencies, dep)
	}

	return dependencies, errors
}

// parseSpecDependency converts a single spec dependency to a domain RepoDependency.
func (r *Resolver) parseSpecDependency(specDep SpecDependency) (*RepoDependency, error) {
	// Validate repo path
	if specDep.Repo == "" {
		return nil, fmt.Errorf("dependency repo path is required")
	}

	// Parse and validate dependency type
	depType, valid := ParseDependencyType(specDep.Type)
	if !valid {
		return nil, fmt.Errorf("invalid dependency type '%s' for repo '%s'", specDep.Type, specDep.Repo)
	}

	// Resolve the target repo path
	targetPath := r.resolveRepoPath(specDep.Repo)

	// Check for self-dependency
	if r.isSameRepo(targetPath) {
		return nil, ErrSelfDependency
	}

	dep := NewRepoDependency(r.rootPath, targetPath, depType)
	if specDep.Description != "" {
		dep.WithDescription(specDep.Description)
	}
	if len(specDep.Features) > 0 {
		dep.WithFeatures(specDep.Features...)
	}

	return dep, nil
}

// resolveRepoPath resolves a repo path relative to the root path.
func (r *Resolver) resolveRepoPath(repoPath string) string {
	if filepath.IsAbs(repoPath) {
		return repoPath
	}
	// Resolve relative path from root
	resolved := filepath.Join(r.rootPath, repoPath)
	// Clean the path to normalize any .. components
	cleaned := filepath.Clean(resolved)
	return cleaned
}

// isSameRepo checks if the target path is the same as the root path.
func (r *Resolver) isSameRepo(targetPath string) bool {
	rootAbs, _ := filepath.Abs(r.rootPath)
	targetAbs, _ := filepath.Abs(targetPath)
	return rootAbs == targetAbs
}

// ValidateDependencyPath checks if a dependency path exists and is a valid repo.
func (r *Resolver) ValidateDependencyPath(repoPath string) error {
	resolved := r.resolveRepoPath(repoPath)

	// Check if path exists
	info, err := os.Stat(resolved)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrRepoNotReachable, resolved)
	}
	if err != nil {
		return fmt.Errorf("error checking path %s: %w", resolved, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dependency path is not a directory: %s", resolved)
	}

	// Check if it's a roady-initialized repo (has .roady directory)
	roadyPath := filepath.Join(resolved, ".roady")
	if _, err := os.Stat(roadyPath); os.IsNotExist(err) {
		// Not a roady repo, but still valid as a dependency
		return nil
	}

	return nil
}

// ValidateAllDependencies validates all dependencies in a list.
func (r *Resolver) ValidateAllDependencies(deps []*RepoDependency) map[string]error {
	errors := make(map[string]error)
	for _, dep := range deps {
		if err := r.ValidateDependencyPath(dep.TargetRepo); err != nil {
			errors[dep.ID] = err
		}
	}
	return errors
}

// ResolveDependencyGraph builds a dependency graph from spec dependencies.
func (r *Resolver) ResolveDependencyGraph(specDeps []SpecDependency) (*DependencyGraph, []error) {
	graph := NewDependencyGraph(r.rootPath)
	deps, parseErrors := r.ParseSpecDependencies(specDeps)

	for _, dep := range deps {
		graph.AddDependency(dep)
	}

	return graph, parseErrors
}

// DiscoverRoadyRepos looks for roady-initialized repos in the given directory.
func (r *Resolver) DiscoverRoadyRepos(searchPath string, maxDepth int) ([]string, error) {
	var repos []string
	baseDepth := len(filepath.SplitList(searchPath))

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		// Check depth limit
		currentDepth := len(filepath.SplitList(path))
		if maxDepth > 0 && currentDepth-baseDepth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for .roady directory
		if info.IsDir() && info.Name() == ".roady" {
			parentDir := filepath.Dir(path)
			repos = append(repos, parentDir)
		}

		return nil
	})

	return repos, err
}

// SuggestDependencies analyzes repo content to suggest potential dependencies.
// This is a placeholder for future implementation that could analyze:
// - go.mod for Go module dependencies
// - package.json for Node.js dependencies
// - Import statements for cross-repo references
func (r *Resolver) SuggestDependencies() ([]*RepoDependency, error) {
	// Placeholder - could be extended to analyze:
	// 1. Go imports looking for local module paths
	// 2. Config files referencing other services
	// 3. Docker compose files for service dependencies
	return nil, nil
}

// DependencyValidationResult holds the result of dependency validation.
type DependencyValidationResult struct {
	Dependency *RepoDependency
	IsValid    bool
	Error      error
	IsRoadyRepo bool
}

// ValidateDependenciesWithDetails validates dependencies and returns detailed results.
func (r *Resolver) ValidateDependenciesWithDetails(deps []*RepoDependency) []DependencyValidationResult {
	results := make([]DependencyValidationResult, len(deps))

	for i, dep := range deps {
		results[i] = DependencyValidationResult{
			Dependency: dep,
			IsValid:    true,
		}

		if err := r.ValidateDependencyPath(dep.TargetRepo); err != nil {
			results[i].IsValid = false
			results[i].Error = err
		}

		// Check if it's a roady repo
		roadyPath := filepath.Join(dep.TargetRepo, ".roady")
		if _, err := os.Stat(roadyPath); err == nil {
			results[i].IsRoadyRepo = true
		}
	}

	return results
}
