package application

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
	"github.com/felixgeelhaar/roady/pkg/domain/dependency"
)

// mockDependencyRepo implements DependencyRepository for testing.
type mockDependencyRepo struct {
	graph *dependency.DependencyGraph
}

func newMockDependencyRepo(rootPath string) *mockDependencyRepo {
	return &mockDependencyRepo{
		graph: dependency.NewDependencyGraph(rootPath),
	}
}

func (m *mockDependencyRepo) SaveDependencyGraph(graph *dependency.DependencyGraph) error {
	m.graph = graph
	return nil
}

func (m *mockDependencyRepo) LoadDependencyGraph() (*dependency.DependencyGraph, error) {
	return m.graph, nil
}

func (m *mockDependencyRepo) AddDependency(dep *dependency.RepoDependency) error {
	m.graph.AddDependency(dep)
	return nil
}

func (m *mockDependencyRepo) RemoveDependency(depID string) error {
	if !m.graph.RemoveDependency(depID) {
		return dependency.ErrDependencyNotFound
	}
	return nil
}

func (m *mockDependencyRepo) GetDependency(depID string) (*dependency.RepoDependency, error) {
	return m.graph.GetDependency(depID), nil
}

func (m *mockDependencyRepo) ListDependencies() ([]*dependency.RepoDependency, error) {
	return m.graph.Dependencies, nil
}

func (m *mockDependencyRepo) UpdateRepoHealth(health *dependency.RepoHealth) error {
	m.graph.SetRepoHealth(health)
	return nil
}

func (m *mockDependencyRepo) GetRepoHealth(repoPath string) (*dependency.RepoHealth, error) {
	return m.graph.GetRepoHealth(repoPath), nil
}

// mockHealthScanner for testing.
type mockHealthScanner struct {
	healthMap map[string]*dependency.RepoHealth
}

func newMockHealthScanner() *mockHealthScanner {
	return &mockHealthScanner{
		healthMap: make(map[string]*dependency.RepoHealth),
	}
}

func (m *mockHealthScanner) ScanRepoHealth(repoPath string) (*dependency.RepoHealth, error) {
	if health, ok := m.healthMap[repoPath]; ok {
		return health, nil
	}
	return &dependency.RepoHealth{
		RepoPath:    repoPath,
		IsReachable: true,
		LastChecked: time.Now(),
	}, nil
}

func TestNewDependencyService(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	svc := NewDependencyService(repo, "/test/root")

	if svc.rootPath != "/test/root" {
		t.Errorf("rootPath = %s, want /test/root", svc.rootPath)
	}
}

func TestDependencyService_GetDependencyGraph(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	svc := NewDependencyService(repo, "/test/root")

	graph, err := svc.GetDependencyGraph()
	if err != nil {
		t.Fatalf("GetDependencyGraph failed: %v", err)
	}
	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}
	if graph.RootRepo != "/test/root" {
		t.Errorf("RootRepo = %s, want /test/root", graph.RootRepo)
	}
}

func TestDependencyService_AddDependency_InvalidType(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	svc := NewDependencyService(repo, "/test/root")

	_, err := svc.AddDependency("/other/repo", dependency.DependencyType("invalid"), "")
	if err != dependency.ErrInvalidDependencyType {
		t.Errorf("Expected ErrInvalidDependencyType, got %v", err)
	}
}

func TestDependencyService_ListDependencies(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	dep := dependency.NewRepoDependency("/test/root", "/other/repo", dependency.DependencyRuntime)
	_ = repo.AddDependency(dep)

	svc := NewDependencyService(repo, "/test/root")

	deps, err := svc.ListDependencies()
	if err != nil {
		t.Fatalf("ListDependencies failed: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	}
}

func TestDependencyService_RemoveDependency(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	dep := dependency.NewRepoDependency("/test/root", "/other/repo", dependency.DependencyRuntime)
	_ = repo.AddDependency(dep)

	svc := NewDependencyService(repo, "/test/root")

	err := svc.RemoveDependency(dep.ID)
	if err != nil {
		t.Fatalf("RemoveDependency failed: %v", err)
	}

	deps, _ := svc.ListDependencies()
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies after removal, got %d", len(deps))
	}
}

func TestDependencyService_RemoveDependency_NotFound(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	svc := NewDependencyService(repo, "/test/root")

	err := svc.RemoveDependency("nonexistent")
	if err != dependency.ErrDependencyNotFound {
		t.Errorf("Expected ErrDependencyNotFound, got %v", err)
	}
}

func TestDependencyService_GetDependency(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	dep := dependency.NewRepoDependency("/test/root", "/other/repo", dependency.DependencyRuntime)
	_ = repo.AddDependency(dep)

	svc := NewDependencyService(repo, "/test/root")

	found, err := svc.GetDependency(dep.ID)
	if err != nil {
		t.Fatalf("GetDependency failed: %v", err)
	}
	if found == nil {
		t.Fatal("Expected dependency to be found")
	}
	if found.ID != dep.ID {
		t.Errorf("ID = %s, want %s", found.ID, dep.ID)
	}
}

func TestDependencyService_ScanDependentRepos(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	_ = repo.AddDependency(dependency.NewRepoDependency("/test/root", "/dep1", dependency.DependencyRuntime))
	_ = repo.AddDependency(dependency.NewRepoDependency("/test/root", "/dep2", dependency.DependencyData))

	scanner := newMockHealthScanner()
	scanner.healthMap["/dep1"] = &dependency.RepoHealth{
		RepoPath:    "/dep1",
		IsReachable: true,
		HasDrift:    false,
	}
	scanner.healthMap["/dep2"] = &dependency.RepoHealth{
		RepoPath:    "/dep2",
		IsReachable: true,
		HasDrift:    true, // Unhealthy
	}

	svc := NewDependencyService(repo, "/test/root")

	result, err := svc.ScanDependentRepos(scanner)
	if err != nil {
		t.Fatalf("ScanDependentRepos failed: %v", err)
	}

	if result.TotalRepos != 3 { // root + 2 deps
		t.Errorf("TotalRepos = %d, want 3", result.TotalRepos)
	}
	if result.AllHealthy() {
		t.Error("Expected not all healthy due to drift")
	}
}

func TestDependencyService_ScanDependentRepos_NoScanner(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	_ = repo.AddDependency(dependency.NewRepoDependency("/test/root", "/dep1", dependency.DependencyRuntime))

	svc := NewDependencyService(repo, "/test/root")

	result, err := svc.ScanDependentRepos(nil) // No scanner
	if err != nil {
		t.Fatalf("ScanDependentRepos failed: %v", err)
	}

	// Should still work with basic health check
	if result.TotalRepos != 2 {
		t.Errorf("TotalRepos = %d, want 2", result.TotalRepos)
	}
}

func TestDependencyService_ImportFromSpec(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	svc := NewDependencyService(repo, "/test/root")

	specDeps := []dependency.SpecDependency{
		{Repo: "/dep1", Type: "runtime", Description: "API"},
		{Repo: "/dep2", Type: "data"},
	}

	result, err := svc.ImportFromSpec(specDeps)
	if err != nil {
		t.Fatalf("ImportFromSpec failed: %v", err)
	}

	if result.Imported != 2 {
		t.Errorf("Imported = %d, want 2", result.Imported)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", result.Skipped)
	}
}

func TestDependencyService_ImportFromSpec_InvalidType(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	svc := NewDependencyService(repo, "/test/root")

	specDeps := []dependency.SpecDependency{
		{Repo: "/dep1", Type: "invalid"},
	}

	result, err := svc.ImportFromSpec(specDeps)
	if err != nil {
		t.Fatalf("ImportFromSpec failed: %v", err)
	}

	if result.Imported != 0 {
		t.Errorf("Imported = %d, want 0", result.Imported)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
}

func TestDependencyService_GetDependencySummary(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	_ = repo.AddDependency(dependency.NewRepoDependency("/test/root", "/dep1", dependency.DependencyRuntime))
	_ = repo.AddDependency(dependency.NewRepoDependency("/test/root", "/dep2", dependency.DependencyRuntime))
	_ = repo.AddDependency(dependency.NewRepoDependency("/test/root", "/dep3", dependency.DependencyData))

	svc := NewDependencyService(repo, "/test/root")

	summary, err := svc.GetDependencySummary()
	if err != nil {
		t.Fatalf("GetDependencySummary failed: %v", err)
	}

	if summary.TotalDependencies != 3 {
		t.Errorf("TotalDependencies = %d, want 3", summary.TotalDependencies)
	}
	if summary.ByType[dependency.DependencyRuntime] != 2 {
		t.Errorf("Runtime deps = %d, want 2", summary.ByType[dependency.DependencyRuntime])
	}
}

func TestDependencyService_CheckForCycles(t *testing.T) {
	repo := newMockDependencyRepo("/a")
	_ = repo.AddDependency(dependency.NewRepoDependency("/a", "/b", dependency.DependencyRuntime))
	_ = repo.AddDependency(dependency.NewRepoDependency("/b", "/c", dependency.DependencyRuntime))

	svc := NewDependencyService(repo, "/a")

	hasCycle, err := svc.CheckForCycles()
	if err != nil {
		t.Fatalf("CheckForCycles failed: %v", err)
	}
	if hasCycle {
		t.Error("Expected no cycle")
	}
}

func TestDependencyService_CheckForCycles_WithCycle(t *testing.T) {
	repo := newMockDependencyRepo("/a")
	_ = repo.AddDependency(dependency.NewRepoDependency("/a", "/b", dependency.DependencyRuntime))
	_ = repo.AddDependency(dependency.NewRepoDependency("/b", "/a", dependency.DependencyRuntime)) // Cycle

	svc := NewDependencyService(repo, "/a")

	hasCycle, err := svc.CheckForCycles()
	if err != nil {
		t.Fatalf("CheckForCycles failed: %v", err)
	}
	if !hasCycle {
		t.Error("Expected cycle to be detected")
	}
}

func TestDependencyService_GetDependencyOrder(t *testing.T) {
	repo := newMockDependencyRepo("/root")
	_ = repo.AddDependency(dependency.NewRepoDependency("/root", "/a", dependency.DependencyRuntime))
	_ = repo.AddDependency(dependency.NewRepoDependency("/a", "/b", dependency.DependencyRuntime))

	svc := NewDependencyService(repo, "/root")

	order, err := svc.GetDependencyOrder()
	if err != nil {
		t.Fatalf("GetDependencyOrder failed: %v", err)
	}
	if len(order) != 3 {
		t.Errorf("Order length = %d, want 3", len(order))
	}
}

func TestDependencyService_GetUnhealthyDependencies(t *testing.T) {
	repo := newMockDependencyRepo("/root")
	_ = repo.UpdateRepoHealth(&dependency.RepoHealth{RepoPath: "/healthy", IsReachable: true})
	_ = repo.UpdateRepoHealth(&dependency.RepoHealth{RepoPath: "/unhealthy", IsReachable: true, HasDrift: true})

	svc := NewDependencyService(repo, "/root")

	unhealthy, err := svc.GetUnhealthyDependencies()
	if err != nil {
		t.Fatalf("GetUnhealthyDependencies failed: %v", err)
	}
	if len(unhealthy) != 1 {
		t.Errorf("Unhealthy count = %d, want 1", len(unhealthy))
	}
}

func TestScanResult_AllHealthy(t *testing.T) {
	tests := []struct {
		name     string
		result   ScanResult
		expected bool
	}{
		{"all healthy", ScanResult{HealthyRepos: 5, UnhealthyRepos: 0, Unreachable: 0}, true},
		{"has unhealthy", ScanResult{HealthyRepos: 3, UnhealthyRepos: 2, Unreachable: 0}, false},
		{"has unreachable", ScanResult{HealthyRepos: 4, UnhealthyRepos: 0, Unreachable: 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.AllHealthy(); got != tt.expected {
				t.Errorf("AllHealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDependencyService_basicHealthCheck(t *testing.T) {
	repo := newMockDependencyRepo("/test/root")
	svc := NewDependencyService(repo, "/test/root")

	health := svc.basicHealthCheck("/nonexistent/path")

	if health.IsReachable {
		t.Error("Expected unreachable for non-existent path")
	}
	if health.VelocityTrend != "" && health.VelocityTrend != analytics.TrendStable {
		// If not reachable, trend may not be set
	}
}
