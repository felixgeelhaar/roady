package dependency

import (
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
)

// DependencyType represents the nature of a dependency between repos.
type DependencyType string

const (
	// DependencyRuntime indicates a runtime/API dependency.
	DependencyRuntime DependencyType = "runtime"
	// DependencyData indicates a data/schema dependency.
	DependencyData DependencyType = "data"
	// DependencyBuild indicates a build-time dependency.
	DependencyBuild DependencyType = "build"
	// DependencyIntent indicates a feature/intent dependency (shared goals).
	DependencyIntent DependencyType = "intent"
)

// AllDependencyTypes returns all valid dependency types.
func AllDependencyTypes() []DependencyType {
	return []DependencyType{
		DependencyRuntime,
		DependencyData,
		DependencyBuild,
		DependencyIntent,
	}
}

// IsValid checks if the dependency type is valid.
func (dt DependencyType) IsValid() bool {
	switch dt {
	case DependencyRuntime, DependencyData, DependencyBuild, DependencyIntent:
		return true
	default:
		return false
	}
}

// ParseDependencyType parses a string into a DependencyType.
func ParseDependencyType(s string) (DependencyType, bool) {
	dt := DependencyType(s)
	return dt, dt.IsValid()
}

// RepoDependency represents a dependency relationship between two repositories.
type RepoDependency struct {
	// ID uniquely identifies this dependency relationship.
	ID string `json:"id" yaml:"id"`
	// SourceRepo is the repo that depends on another.
	SourceRepo string `json:"source_repo" yaml:"source_repo"`
	// TargetRepo is the repo being depended on.
	TargetRepo string `json:"target_repo" yaml:"target_repo"`
	// Type indicates the nature of the dependency.
	Type DependencyType `json:"type" yaml:"type"`
	// Description provides context about this dependency.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// FeatureIDs lists features that establish this dependency.
	FeatureIDs []string `json:"feature_ids,omitempty" yaml:"feature_ids,omitempty"`
	// CreatedAt is when this dependency was first recorded.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	// UpdatedAt is when this dependency was last updated.
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// NewRepoDependency creates a new repository dependency.
func NewRepoDependency(source, target string, depType DependencyType) *RepoDependency {
	now := time.Now()
	return &RepoDependency{
		ID:         generateDependencyID(source, target, depType),
		SourceRepo: source,
		TargetRepo: target,
		Type:       depType,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// generateDependencyID creates a unique ID for a dependency.
func generateDependencyID(source, target string, depType DependencyType) string {
	return source + "->" + target + ":" + string(depType)
}

// WithDescription adds a description to the dependency.
func (d *RepoDependency) WithDescription(desc string) *RepoDependency {
	d.Description = desc
	d.UpdatedAt = time.Now()
	return d
}

// WithFeatures adds feature IDs to the dependency.
func (d *RepoDependency) WithFeatures(featureIDs ...string) *RepoDependency {
	d.FeatureIDs = append(d.FeatureIDs, featureIDs...)
	d.UpdatedAt = time.Now()
	return d
}

// RepoHealth represents the health status of a repository.
type RepoHealth struct {
	// RepoPath is the path to the repository.
	RepoPath string `json:"repo_path"`
	// HasDrift indicates if there is drift from the spec.
	HasDrift bool `json:"has_drift"`
	// DriftCount is the number of drift items detected.
	DriftCount int `json:"drift_count"`
	// VelocityTrend indicates the velocity direction.
	VelocityTrend analytics.TrendDirection `json:"velocity_trend"`
	// CompletionRate is the percentage of completed tasks.
	CompletionRate float64 `json:"completion_rate"`
	// LastChecked is when the repo was last analyzed.
	LastChecked time.Time `json:"last_checked"`
	// IsReachable indicates if the repo path is accessible.
	IsReachable bool `json:"is_reachable"`
	// ErrorMessage contains any error encountered during health check.
	ErrorMessage string `json:"error_message,omitempty"`
}

// IsHealthy returns true if the repo has no drift and positive velocity.
func (h *RepoHealth) IsHealthy() bool {
	return h.IsReachable && !h.HasDrift && h.VelocityTrend != analytics.TrendDecelerating
}

// HealthStatus returns a human-readable health status.
func (h *RepoHealth) HealthStatus() string {
	if !h.IsReachable {
		return "unreachable"
	}
	if h.HasDrift {
		return "drifting"
	}
	if h.VelocityTrend == analytics.TrendDecelerating {
		return "slowing"
	}
	if h.VelocityTrend == analytics.TrendAccelerating {
		return "accelerating"
	}
	return "stable"
}

// DependencyGraph represents the full dependency graph for a workspace.
type DependencyGraph struct {
	// RootRepo is the main repository being analyzed.
	RootRepo string `json:"root_repo"`
	// Dependencies are all tracked dependencies.
	Dependencies []*RepoDependency `json:"dependencies"`
	// RepoHealth maps repo paths to their health status.
	RepoHealth map[string]*RepoHealth `json:"repo_health,omitempty"`
	// LastUpdated is when the graph was last updated.
	LastUpdated time.Time `json:"last_updated"`
}

// NewDependencyGraph creates a new dependency graph.
func NewDependencyGraph(rootRepo string) *DependencyGraph {
	return &DependencyGraph{
		RootRepo:     rootRepo,
		Dependencies: make([]*RepoDependency, 0),
		RepoHealth:   make(map[string]*RepoHealth),
		LastUpdated:  time.Now(),
	}
}

// AddDependency adds a dependency to the graph.
func (g *DependencyGraph) AddDependency(dep *RepoDependency) {
	// Check for duplicates
	for i, existing := range g.Dependencies {
		if existing.ID == dep.ID {
			g.Dependencies[i] = dep
			g.LastUpdated = time.Now()
			return
		}
	}
	g.Dependencies = append(g.Dependencies, dep)
	g.LastUpdated = time.Now()
}

// RemoveDependency removes a dependency by ID.
func (g *DependencyGraph) RemoveDependency(id string) bool {
	for i, dep := range g.Dependencies {
		if dep.ID == id {
			g.Dependencies = append(g.Dependencies[:i], g.Dependencies[i+1:]...)
			g.LastUpdated = time.Now()
			return true
		}
	}
	return false
}

// GetDependency retrieves a dependency by ID.
func (g *DependencyGraph) GetDependency(id string) *RepoDependency {
	for _, dep := range g.Dependencies {
		if dep.ID == id {
			return dep
		}
	}
	return nil
}

// GetDependenciesForRepo returns all dependencies where the given repo is the source.
func (g *DependencyGraph) GetDependenciesForRepo(repoPath string) []*RepoDependency {
	result := make([]*RepoDependency, 0)
	for _, dep := range g.Dependencies {
		if dep.SourceRepo == repoPath {
			result = append(result, dep)
		}
	}
	return result
}

// GetDependentsOfRepo returns all repos that depend on the given repo.
func (g *DependencyGraph) GetDependentsOfRepo(repoPath string) []*RepoDependency {
	result := make([]*RepoDependency, 0)
	for _, dep := range g.Dependencies {
		if dep.TargetRepo == repoPath {
			result = append(result, dep)
		}
	}
	return result
}

// GetDependenciesByType returns all dependencies of a specific type.
func (g *DependencyGraph) GetDependenciesByType(depType DependencyType) []*RepoDependency {
	result := make([]*RepoDependency, 0)
	for _, dep := range g.Dependencies {
		if dep.Type == depType {
			result = append(result, dep)
		}
	}
	return result
}

// GetAllRepos returns all unique repo paths in the graph.
func (g *DependencyGraph) GetAllRepos() []string {
	repos := make(map[string]struct{})
	repos[g.RootRepo] = struct{}{}

	for _, dep := range g.Dependencies {
		repos[dep.SourceRepo] = struct{}{}
		repos[dep.TargetRepo] = struct{}{}
	}

	result := make([]string, 0, len(repos))
	for repo := range repos {
		result = append(result, repo)
	}
	return result
}

// SetRepoHealth updates the health status for a repository.
func (g *DependencyGraph) SetRepoHealth(health *RepoHealth) {
	g.RepoHealth[health.RepoPath] = health
	g.LastUpdated = time.Now()
}

// GetRepoHealth retrieves the health status for a repository.
func (g *DependencyGraph) GetRepoHealth(repoPath string) *RepoHealth {
	return g.RepoHealth[repoPath]
}

// GetUnhealthyRepos returns all repos with health issues.
func (g *DependencyGraph) GetUnhealthyRepos() []*RepoHealth {
	result := make([]*RepoHealth, 0)
	for _, health := range g.RepoHealth {
		if !health.IsHealthy() {
			result = append(result, health)
		}
	}
	return result
}

// HasCycle checks if the dependency graph has any cycles.
func (g *DependencyGraph) HasCycle() bool {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var dfs func(repo string) bool
	dfs = func(repo string) bool {
		visited[repo] = true
		inStack[repo] = true

		for _, dep := range g.GetDependenciesForRepo(repo) {
			target := dep.TargetRepo
			if !visited[target] {
				if dfs(target) {
					return true
				}
			} else if inStack[target] {
				return true
			}
		}

		inStack[repo] = false
		return false
	}

	for _, repo := range g.GetAllRepos() {
		if !visited[repo] {
			if dfs(repo) {
				return true
			}
		}
	}

	return false
}

// TopologicalSort returns repos in dependency order (dependencies first).
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	if g.HasCycle() {
		return nil, ErrCyclicDependency
	}

	visited := make(map[string]bool)
	result := make([]string, 0)

	var visit func(repo string)
	visit = func(repo string) {
		if visited[repo] {
			return
		}
		visited[repo] = true

		// Visit dependencies first
		for _, dep := range g.GetDependenciesForRepo(repo) {
			visit(dep.TargetRepo)
		}

		result = append(result, repo)
	}

	// Start from all repos (to handle disconnected components)
	for _, repo := range g.GetAllRepos() {
		visit(repo)
	}

	return result, nil
}

// DependencySummary provides a summary of the dependency graph.
type DependencySummary struct {
	TotalDependencies int                    `json:"total_dependencies"`
	TotalRepos        int                    `json:"total_repos"`
	ByType            map[DependencyType]int `json:"by_type"`
	UnhealthyCount    int                    `json:"unhealthy_count"`
	HasCycles         bool                   `json:"has_cycles"`
}

// GetSummary returns a summary of the dependency graph.
func (g *DependencyGraph) GetSummary() DependencySummary {
	summary := DependencySummary{
		TotalDependencies: len(g.Dependencies),
		TotalRepos:        len(g.GetAllRepos()),
		ByType:            make(map[DependencyType]int),
		UnhealthyCount:    len(g.GetUnhealthyRepos()),
		HasCycles:         g.HasCycle(),
	}

	for _, dep := range g.Dependencies {
		summary.ByType[dep.Type]++
	}

	return summary
}
