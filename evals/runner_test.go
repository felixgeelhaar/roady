// Package evals contains regression tests over Roady's planning pipeline.
//
// Each fixture under fixtures/ pairs a spec.yaml with a golden.yaml describing
// the exact set of tasks (IDs, feature IDs, dependency edges) the heuristic
// planner is expected to produce. When the planner contract changes
// intentionally, regenerate the goldens; an unintentional change will fail
// these tests and surface the behavior diff.
package evals

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

type goldenTask struct {
	ID        string   `yaml:"id"`
	FeatureID string   `yaml:"feature_id"`
	DependsOn []string `yaml:"depends_on"`
}

type goldenPlan struct {
	SpecID string       `yaml:"spec_id"`
	Tasks  []goldenTask `yaml:"tasks"`
}

func TestHeuristicPlannerMatchesGoldens(t *testing.T) {
	fixturesDir, err := filepath.Abs("fixtures")
	if err != nil {
		t.Fatalf("resolve fixtures dir: %v", err)
	}

	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no fixtures discovered; expected at least one fixture directory")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			runFixture(t, filepath.Join(fixturesDir, name))
		})
	}
}

func runFixture(t *testing.T, fixtureDir string) {
	t.Helper()

	loadedSpec := loadSpec(t, filepath.Join(fixtureDir, "spec.yaml"))
	golden := loadGolden(t, filepath.Join(fixtureDir, "golden.yaml"))

	repoRoot := t.TempDir()
	repo := storage.NewFilesystemRepository(repoRoot)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialise repo: %v", err)
	}
	if err := repo.SaveSpec(loadedSpec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)

	plan, err := planSvc.GeneratePlan(context.Background())
	if err != nil {
		t.Fatalf("generate plan: %v", err)
	}

	if plan.SpecID != golden.SpecID {
		t.Errorf("plan.SpecID = %q, want %q", plan.SpecID, golden.SpecID)
	}

	if len(plan.Tasks) != len(golden.Tasks) {
		t.Fatalf("task count = %d, want %d\n  got tasks:    %s\n  golden tasks: %s",
			len(plan.Tasks), len(golden.Tasks),
			summariseTasks(plan.Tasks), summariseGolden(golden.Tasks))
	}

	gotByID := make(map[string]goldenTask, len(plan.Tasks))
	for _, task := range plan.Tasks {
		deps := append([]string(nil), task.DependsOn...)
		sort.Strings(deps)
		gotByID[task.ID] = goldenTask{
			ID:        task.ID,
			FeatureID: task.FeatureID,
			DependsOn: deps,
		}
	}

	for _, want := range golden.Tasks {
		got, ok := gotByID[want.ID]
		if !ok {
			t.Errorf("missing task %q in generated plan", want.ID)
			continue
		}
		if got.FeatureID != want.FeatureID {
			t.Errorf("task %q feature_id = %q, want %q", want.ID, got.FeatureID, want.FeatureID)
		}
		wantDeps := append([]string(nil), want.DependsOn...)
		sort.Strings(wantDeps)
		if !equalStrings(got.DependsOn, wantDeps) {
			t.Errorf("task %q depends_on = %v, want %v", want.ID, got.DependsOn, wantDeps)
		}
	}

	// Every heuristic-emitted task must carry the heuristic origin so AI
	// planner regressions that silently mix origins surface here.
	for _, task := range plan.Tasks {
		if task.NormalisedOrigin() != "heuristic" {
			t.Errorf("task %q origin = %q, want heuristic", task.ID, task.NormalisedOrigin())
		}
	}
}

type taskSummary struct{ id, feature string }

func summariseTasks(tasks interface{}) string {
	out := []taskSummary{}
	if v, ok := tasks.([]goldenTask); ok {
		for _, t := range v {
			out = append(out, taskSummary{t.ID, t.FeatureID})
		}
	}
	return formatSummaries(out)
}

func summariseGolden(tasks []goldenTask) string {
	out := make([]taskSummary, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, taskSummary{t.ID, t.FeatureID})
	}
	return formatSummaries(out)
}

func formatSummaries(in []taskSummary) string {
	s := ""
	for i, t := range in {
		if i > 0 {
			s += ", "
		}
		s += t.id
	}
	return s
}

func loadSpec(t *testing.T, path string) *spec.ProductSpec {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spec %s: %v", path, err)
	}
	var s spec.ProductSpec
	if err := yaml.Unmarshal(data, &s); err != nil {
		t.Fatalf("decode spec %s: %v", path, err)
	}
	return &s
}

func loadGolden(t *testing.T, path string) goldenPlan {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	var g goldenPlan
	if err := yaml.Unmarshal(data, &g); err != nil {
		t.Fatalf("decode golden %s: %v", path, err)
	}
	return g
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
