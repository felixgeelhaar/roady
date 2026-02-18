package application

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
	"github.com/felixgeelhaar/roady/pkg/domain/dependency"
)

// DependencyRepository defines the storage interface for dependency data.
type DependencyRepository interface {
	SaveDependencyGraph(graph *dependency.DependencyGraph) error
	LoadDependencyGraph() (*dependency.DependencyGraph, error)
	AddDependency(dep *dependency.RepoDependency) error
	RemoveDependency(depID string) error
	GetDependency(depID string) (*dependency.RepoDependency, error)
	ListDependencies() ([]*dependency.RepoDependency, error)
	UpdateRepoHealth(health *dependency.RepoHealth) error
	GetRepoHealth(repoPath string) (*dependency.RepoHealth, error)
}

// HealthScanner defines an interface for scanning repo health.
type HealthScanner interface {
	ScanRepoHealth(repoPath string) (*dependency.RepoHealth, error)
}

// DependencyService manages cross-repo dependencies.
type DependencyService struct {
	repo     DependencyRepository
	resolver *dependency.Resolver
	rootPath string
}

// NewDependencyService creates a new dependency service.
func NewDependencyService(repo DependencyRepository, rootPath string) *DependencyService {
	return &DependencyService{
		repo:     repo,
		resolver: dependency.NewResolver(rootPath),
		rootPath: rootPath,
	}
}

// GetDependencyGraph returns the current dependency graph.
func (s *DependencyService) GetDependencyGraph() (*dependency.DependencyGraph, error) {
	graph, err := s.repo.LoadDependencyGraph()
	if err != nil {
		return nil, err
	}
	if graph == nil {
		return dependency.NewDependencyGraph(s.rootPath), nil
	}
	return graph, nil
}

// AddDependency adds a new dependency to the graph.
func (s *DependencyService) AddDependency(targetRepo string, depType dependency.DependencyType, description string) (*dependency.RepoDependency, error) {
	// Validate the dependency type
	if !depType.IsValid() {
		return nil, dependency.ErrInvalidDependencyType
	}

	// Create the dependency
	dep := dependency.NewRepoDependency(s.rootPath, targetRepo, depType)
	if description != "" {
		dep.WithDescription(description)
	}

	// Validate the target path exists
	if err := s.resolver.ValidateDependencyPath(targetRepo); err != nil {
		return nil, err
	}

	// Check for self-dependency
	if s.resolver.ValidateDependencyPath(".") == nil {
		// Path resolves - check if it's the same
		resolvedRoot := s.resolver.ValidateDependencyPath(s.rootPath)
		resolvedTarget := s.resolver.ValidateDependencyPath(targetRepo)
		if resolvedRoot == resolvedTarget {
			return nil, dependency.ErrSelfDependency
		}
	}

	// Add to storage
	if err := s.repo.AddDependency(dep); err != nil {
		return nil, fmt.Errorf("failed to add dependency: %w", err)
	}

	return dep, nil
}

// RemoveDependency removes a dependency by ID.
func (s *DependencyService) RemoveDependency(depID string) error {
	return s.repo.RemoveDependency(depID)
}

// ListDependencies returns all dependencies.
func (s *DependencyService) ListDependencies() ([]*dependency.RepoDependency, error) {
	return s.repo.ListDependencies()
}

// GetDependency retrieves a specific dependency.
func (s *DependencyService) GetDependency(depID string) (*dependency.RepoDependency, error) {
	return s.repo.GetDependency(depID)
}

// ScanDependentRepos scans health status of all dependent repositories.
func (s *DependencyService) ScanDependentRepos(healthScanner HealthScanner) (*ScanResult, error) {
	graph, err := s.GetDependencyGraph()
	if err != nil {
		return nil, err
	}

	result := &ScanResult{
		ScannedAt:     time.Now(),
		TotalRepos:    0,
		HealthyRepos:  0,
		UnhealthyRepos: 0,
		Unreachable:   0,
		Details:       make(map[string]*dependency.RepoHealth),
	}

	// Get all unique repos
	repos := graph.GetAllRepos()
	result.TotalRepos = len(repos)

	for _, repoPath := range repos {
		if repoPath == s.rootPath {
			continue // Skip self
		}

		var health *dependency.RepoHealth
		if healthScanner != nil {
			health, err = healthScanner.ScanRepoHealth(repoPath)
			if err != nil {
				health = &dependency.RepoHealth{
					RepoPath:     repoPath,
					IsReachable:  false,
					ErrorMessage: err.Error(),
					LastChecked:  time.Now(),
				}
			}
		} else {
			// Basic health check without scanner
			health = s.basicHealthCheck(repoPath)
		}

		result.Details[repoPath] = health

		switch {
		case !health.IsReachable:
			result.Unreachable++
		case health.IsHealthy():
			result.HealthyRepos++
		default:
			result.UnhealthyRepos++
		}

		// Update stored health
		_ = s.repo.UpdateRepoHealth(health)
	}

	return result, nil
}

// basicHealthCheck performs a simple health check without external scanner.
func (s *DependencyService) basicHealthCheck(repoPath string) *dependency.RepoHealth {
	health := &dependency.RepoHealth{
		RepoPath:    repoPath,
		LastChecked: time.Now(),
	}

	// Check if path is reachable
	if err := s.resolver.ValidateDependencyPath(repoPath); err != nil {
		health.IsReachable = false
		health.ErrorMessage = err.Error()
		return health
	}

	health.IsReachable = true
	health.VelocityTrend = analytics.TrendStable
	return health
}

// ImportFromSpec imports dependencies from spec configuration.
func (s *DependencyService) ImportFromSpec(specDeps []dependency.SpecDependency) (*ImportResult, error) {
	result := &ImportResult{
		Imported: 0,
		Skipped:  0,
		Errors:   make(map[string]string),
	}

	deps, parseErrors := s.resolver.ParseSpecDependencies(specDeps)

	for _, err := range parseErrors {
		result.Errors["parse"] = err.Error()
		result.Skipped++
	}

	for _, dep := range deps {
		if err := s.repo.AddDependency(dep); err != nil {
			result.Errors[dep.ID] = err.Error()
			result.Skipped++
		} else {
			result.Imported++
		}
	}

	return result, nil
}

// GetDependencySummary returns a summary of the dependency graph.
func (s *DependencyService) GetDependencySummary() (*dependency.DependencySummary, error) {
	graph, err := s.GetDependencyGraph()
	if err != nil {
		return nil, err
	}

	summary := graph.GetSummary()
	return &summary, nil
}

// CheckForCycles checks if the dependency graph has cycles.
func (s *DependencyService) CheckForCycles() (bool, error) {
	graph, err := s.GetDependencyGraph()
	if err != nil {
		return false, err
	}
	return graph.HasCycle(), nil
}

// GetDependencyOrder returns repos in dependency order (dependencies first).
func (s *DependencyService) GetDependencyOrder() ([]string, error) {
	graph, err := s.GetDependencyGraph()
	if err != nil {
		return nil, err
	}
	return graph.TopologicalSort()
}

// GetUnhealthyDependencies returns all unhealthy dependencies.
func (s *DependencyService) GetUnhealthyDependencies() ([]*dependency.RepoHealth, error) {
	graph, err := s.GetDependencyGraph()
	if err != nil {
		return nil, err
	}
	return graph.GetUnhealthyRepos(), nil
}

// ScanResult contains the results of a dependency scan.
type ScanResult struct {
	ScannedAt      time.Time                         `json:"scanned_at"`
	TotalRepos     int                               `json:"total_repos"`
	HealthyRepos   int                               `json:"healthy_repos"`
	UnhealthyRepos int                               `json:"unhealthy_repos"`
	Unreachable    int                               `json:"unreachable"`
	Details        map[string]*dependency.RepoHealth `json:"details"`
}

// AllHealthy returns true if all repos are healthy.
func (r *ScanResult) AllHealthy() bool {
	return r.UnhealthyRepos == 0 && r.Unreachable == 0
}

// ImportResult contains the results of importing dependencies from spec.
type ImportResult struct {
	Imported int               `json:"imported"`
	Skipped  int               `json:"skipped"`
	Errors   map[string]string `json:"errors,omitempty"`
}
