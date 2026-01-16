package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain/dependency"
)

// DependenciesFile is the filename for storing dependency graph data.
const DependenciesFile = "dependencies.json"

// SaveDependencyGraph persists the dependency graph to storage.
func (r *FilesystemRepository) SaveDependencyGraph(graph *dependency.DependencyGraph) error {
	path, err := r.ResolvePath(DependenciesFile)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dependency graph: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LoadDependencyGraph loads the dependency graph from storage.
func (r *FilesystemRepository) LoadDependencyGraph() (*dependency.DependencyGraph, error) {
	if _, err := os.Stat(r.root); err != nil {
		return nil, fmt.Errorf("root directory does not exist: %w", err)
	}

	path, err := r.ResolvePath(DependenciesFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return nil, nil to indicate no dependencies exist yet
		}
		return nil, fmt.Errorf("failed to read dependencies file: %w", err)
	}

	var graph dependency.DependencyGraph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependency graph: %w", err)
	}

	return &graph, nil
}

// AddDependency adds a single dependency to the stored graph.
func (r *FilesystemRepository) AddDependency(dep *dependency.RepoDependency) error {
	graph, err := r.LoadDependencyGraph()
	if err != nil {
		return err
	}

	if graph == nil {
		graph = dependency.NewDependencyGraph(r.root)
	}

	graph.AddDependency(dep)
	return r.SaveDependencyGraph(graph)
}

// RemoveDependency removes a dependency from the stored graph by ID.
func (r *FilesystemRepository) RemoveDependency(depID string) error {
	graph, err := r.LoadDependencyGraph()
	if err != nil {
		return err
	}

	if graph == nil {
		return dependency.ErrDependencyNotFound
	}

	if !graph.RemoveDependency(depID) {
		return dependency.ErrDependencyNotFound
	}

	return r.SaveDependencyGraph(graph)
}

// GetDependency retrieves a specific dependency by ID.
func (r *FilesystemRepository) GetDependency(depID string) (*dependency.RepoDependency, error) {
	graph, err := r.LoadDependencyGraph()
	if err != nil {
		return nil, err
	}

	if graph == nil {
		return nil, nil
	}

	return graph.GetDependency(depID), nil
}

// ListDependencies returns all dependencies in the graph.
func (r *FilesystemRepository) ListDependencies() ([]*dependency.RepoDependency, error) {
	graph, err := r.LoadDependencyGraph()
	if err != nil {
		return nil, err
	}

	if graph == nil {
		return []*dependency.RepoDependency{}, nil
	}

	return graph.Dependencies, nil
}

// UpdateRepoHealth updates the health status for a repository in the graph.
func (r *FilesystemRepository) UpdateRepoHealth(health *dependency.RepoHealth) error {
	graph, err := r.LoadDependencyGraph()
	if err != nil {
		return err
	}

	if graph == nil {
		graph = dependency.NewDependencyGraph(r.root)
	}

	graph.SetRepoHealth(health)
	return r.SaveDependencyGraph(graph)
}

// GetRepoHealth retrieves the health status for a repository.
func (r *FilesystemRepository) GetRepoHealth(repoPath string) (*dependency.RepoHealth, error) {
	graph, err := r.LoadDependencyGraph()
	if err != nil {
		return nil, err
	}

	if graph == nil {
		return nil, nil
	}

	return graph.GetRepoHealth(repoPath), nil
}
