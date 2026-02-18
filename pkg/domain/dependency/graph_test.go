package dependency

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
)

func TestDependencyType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		depType  DependencyType
		expected bool
	}{
		{"runtime", DependencyRuntime, true},
		{"data", DependencyData, true},
		{"build", DependencyBuild, true},
		{"intent", DependencyIntent, true},
		{"invalid", DependencyType("invalid"), false},
		{"empty", DependencyType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.depType.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseDependencyType(t *testing.T) {
	tests := []struct {
		input    string
		expected DependencyType
		valid    bool
	}{
		{"runtime", DependencyRuntime, true},
		{"data", DependencyData, true},
		{"build", DependencyBuild, true},
		{"intent", DependencyIntent, true},
		{"invalid", DependencyType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, valid := ParseDependencyType(tt.input)
			if valid != tt.valid {
				t.Errorf("ParseDependencyType() valid = %v, want %v", valid, tt.valid)
			}
			if valid && got != tt.expected {
				t.Errorf("ParseDependencyType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewRepoDependency(t *testing.T) {
	dep := NewRepoDependency("./service-a", "../service-b", DependencyRuntime)

	if dep.SourceRepo != "./service-a" {
		t.Errorf("SourceRepo = %s, want ./service-a", dep.SourceRepo)
	}
	if dep.TargetRepo != "../service-b" {
		t.Errorf("TargetRepo = %s, want ../service-b", dep.TargetRepo)
	}
	if dep.Type != DependencyRuntime {
		t.Errorf("Type = %s, want runtime", dep.Type)
	}
	if dep.ID == "" {
		t.Error("ID should not be empty")
	}
	if dep.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestRepoDependency_WithDescription(t *testing.T) {
	dep := NewRepoDependency("./a", "./b", DependencyRuntime)
	dep.WithDescription("test description")

	if dep.Description != "test description" {
		t.Errorf("Description = %s, want 'test description'", dep.Description)
	}
}

func TestRepoDependency_WithFeatures(t *testing.T) {
	dep := NewRepoDependency("./a", "./b", DependencyRuntime)
	dep.WithFeatures("feat-1", "feat-2")

	if len(dep.FeatureIDs) != 2 {
		t.Errorf("FeatureIDs length = %d, want 2", len(dep.FeatureIDs))
	}
	if dep.FeatureIDs[0] != "feat-1" || dep.FeatureIDs[1] != "feat-2" {
		t.Errorf("FeatureIDs = %v, want [feat-1, feat-2]", dep.FeatureIDs)
	}
}

func TestRepoHealth_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		health   RepoHealth
		expected bool
	}{
		{
			"healthy",
			RepoHealth{IsReachable: true, HasDrift: false, VelocityTrend: analytics.TrendStable},
			true,
		},
		{
			"accelerating is healthy",
			RepoHealth{IsReachable: true, HasDrift: false, VelocityTrend: analytics.TrendAccelerating},
			true,
		},
		{
			"not reachable",
			RepoHealth{IsReachable: false, HasDrift: false, VelocityTrend: analytics.TrendStable},
			false,
		},
		{
			"has drift",
			RepoHealth{IsReachable: true, HasDrift: true, VelocityTrend: analytics.TrendStable},
			false,
		},
		{
			"decelerating",
			RepoHealth{IsReachable: true, HasDrift: false, VelocityTrend: analytics.TrendDecelerating},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.health.IsHealthy(); got != tt.expected {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRepoHealth_HealthStatus(t *testing.T) {
	tests := []struct {
		name     string
		health   RepoHealth
		expected string
	}{
		{"unreachable", RepoHealth{IsReachable: false}, "unreachable"},
		{"drifting", RepoHealth{IsReachable: true, HasDrift: true}, "drifting"},
		{"slowing", RepoHealth{IsReachable: true, VelocityTrend: analytics.TrendDecelerating}, "slowing"},
		{"accelerating", RepoHealth{IsReachable: true, VelocityTrend: analytics.TrendAccelerating}, "accelerating"},
		{"stable", RepoHealth{IsReachable: true, VelocityTrend: analytics.TrendStable}, "stable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.health.HealthStatus(); got != tt.expected {
				t.Errorf("HealthStatus() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestDependencyGraph_AddDependency(t *testing.T) {
	g := NewDependencyGraph("./root")
	dep := NewRepoDependency("./root", "./dep1", DependencyRuntime)

	g.AddDependency(dep)

	if len(g.Dependencies) != 1 {
		t.Errorf("Dependencies count = %d, want 1", len(g.Dependencies))
	}

	// Adding same dependency should update, not duplicate
	dep.Description = "updated"
	g.AddDependency(dep)

	if len(g.Dependencies) != 1 {
		t.Errorf("Dependencies count after update = %d, want 1", len(g.Dependencies))
	}
	if g.Dependencies[0].Description != "updated" {
		t.Error("Dependency should be updated")
	}
}

func TestDependencyGraph_RemoveDependency(t *testing.T) {
	g := NewDependencyGraph("./root")
	dep := NewRepoDependency("./root", "./dep1", DependencyRuntime)
	g.AddDependency(dep)

	removed := g.RemoveDependency(dep.ID)
	if !removed {
		t.Error("RemoveDependency should return true")
	}
	if len(g.Dependencies) != 0 {
		t.Error("Dependencies should be empty after removal")
	}

	// Removing non-existent should return false
	removed = g.RemoveDependency("non-existent")
	if removed {
		t.Error("RemoveDependency should return false for non-existent")
	}
}

func TestDependencyGraph_GetDependency(t *testing.T) {
	g := NewDependencyGraph("./root")
	dep := NewRepoDependency("./root", "./dep1", DependencyRuntime)
	g.AddDependency(dep)

	found := g.GetDependency(dep.ID)
	if found == nil {
		t.Fatal("GetDependency should find existing dependency")
	}
	if found.ID != dep.ID {
		t.Errorf("GetDependency returned wrong dependency")
	}

	notFound := g.GetDependency("non-existent")
	if notFound != nil {
		t.Error("GetDependency should return nil for non-existent")
	}
}

func TestDependencyGraph_GetDependenciesForRepo(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./root", "./dep1", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./root", "./dep2", DependencyData))
	g.AddDependency(NewRepoDependency("./other", "./dep3", DependencyBuild))

	deps := g.GetDependenciesForRepo("./root")
	if len(deps) != 2 {
		t.Errorf("GetDependenciesForRepo count = %d, want 2", len(deps))
	}
}

func TestDependencyGraph_GetDependentsOfRepo(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./a", "./shared", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./b", "./shared", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./c", "./other", DependencyRuntime))

	deps := g.GetDependentsOfRepo("./shared")
	if len(deps) != 2 {
		t.Errorf("GetDependentsOfRepo count = %d, want 2", len(deps))
	}
}

func TestDependencyGraph_GetDependenciesByType(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./a", "./b", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./c", "./d", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./e", "./f", DependencyData))

	runtime := g.GetDependenciesByType(DependencyRuntime)
	if len(runtime) != 2 {
		t.Errorf("Runtime dependencies = %d, want 2", len(runtime))
	}

	data := g.GetDependenciesByType(DependencyData)
	if len(data) != 1 {
		t.Errorf("Data dependencies = %d, want 1", len(data))
	}
}

func TestDependencyGraph_GetAllRepos(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./root", "./dep1", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./root", "./dep2", DependencyRuntime))

	repos := g.GetAllRepos()
	if len(repos) != 3 { // root, dep1, dep2
		t.Errorf("GetAllRepos count = %d, want 3", len(repos))
	}
}

func TestDependencyGraph_RepoHealth(t *testing.T) {
	g := NewDependencyGraph("./root")

	health := &RepoHealth{
		RepoPath:    "./root",
		IsReachable: true,
		HasDrift:    false,
	}
	g.SetRepoHealth(health)

	got := g.GetRepoHealth("./root")
	if got == nil {
		t.Fatal("GetRepoHealth should return health")
	}
	if got.RepoPath != "./root" {
		t.Errorf("RepoPath = %s, want ./root", got.RepoPath)
	}

	notFound := g.GetRepoHealth("./not-exists")
	if notFound != nil {
		t.Error("GetRepoHealth should return nil for unknown repo")
	}
}

func TestDependencyGraph_GetUnhealthyRepos(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.SetRepoHealth(&RepoHealth{RepoPath: "./healthy", IsReachable: true})
	g.SetRepoHealth(&RepoHealth{RepoPath: "./drifting", IsReachable: true, HasDrift: true})
	g.SetRepoHealth(&RepoHealth{RepoPath: "./unreachable", IsReachable: false})

	unhealthy := g.GetUnhealthyRepos()
	if len(unhealthy) != 2 {
		t.Errorf("GetUnhealthyRepos count = %d, want 2", len(unhealthy))
	}
}

func TestDependencyGraph_HasCycle_NoCycle(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./root", "./a", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./a", "./b", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./b", "./c", DependencyRuntime))

	if g.HasCycle() {
		t.Error("HasCycle should return false for acyclic graph")
	}
}

func TestDependencyGraph_HasCycle_WithCycle(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./a", "./b", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./b", "./c", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./c", "./a", DependencyRuntime)) // Creates cycle

	if !g.HasCycle() {
		t.Error("HasCycle should return true for cyclic graph")
	}
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./root", "./a", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./a", "./b", DependencyRuntime))

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Errorf("TopologicalSort error: %v", err)
	}
	if len(sorted) != 3 {
		t.Errorf("TopologicalSort length = %d, want 3", len(sorted))
	}

	// ./b should come before ./a, ./a should come before ./root
	bIndex := indexOf(sorted, "./b")
	aIndex := indexOf(sorted, "./a")
	rootIndex := indexOf(sorted, "./root")

	if bIndex > aIndex {
		t.Error("./b should come before ./a in topological order")
	}
	if aIndex > rootIndex {
		t.Error("./a should come before ./root in topological order")
	}
}

func TestDependencyGraph_TopologicalSort_Cycle(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./a", "./b", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./b", "./a", DependencyRuntime)) // Creates cycle

	_, err := g.TopologicalSort()
	if err != ErrCyclicDependency {
		t.Errorf("TopologicalSort should return ErrCyclicDependency, got %v", err)
	}
}

func TestDependencyGraph_GetSummary(t *testing.T) {
	g := NewDependencyGraph("./root")
	g.AddDependency(NewRepoDependency("./root", "./a", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./root", "./b", DependencyRuntime))
	g.AddDependency(NewRepoDependency("./root", "./c", DependencyData))
	g.SetRepoHealth(&RepoHealth{RepoPath: "./unhealthy", IsReachable: false})

	summary := g.GetSummary()

	if summary.TotalDependencies != 3 {
		t.Errorf("TotalDependencies = %d, want 3", summary.TotalDependencies)
	}
	if summary.TotalRepos != 4 { // root, a, b, c
		t.Errorf("TotalRepos = %d, want 4", summary.TotalRepos)
	}
	if summary.ByType[DependencyRuntime] != 2 {
		t.Errorf("ByType[runtime] = %d, want 2", summary.ByType[DependencyRuntime])
	}
	if summary.ByType[DependencyData] != 1 {
		t.Errorf("ByType[data] = %d, want 1", summary.ByType[DependencyData])
	}
	if summary.UnhealthyCount != 1 {
		t.Errorf("UnhealthyCount = %d, want 1", summary.UnhealthyCount)
	}
	if summary.HasCycles {
		t.Error("HasCycles should be false")
	}
}

// Helper function
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
