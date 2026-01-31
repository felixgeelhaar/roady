package application

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/org"
)

func TestOrgService_DiscoverProjects(t *testing.T) {
	root := t.TempDir()

	// Create two project directories with .roady
	for _, name := range []string{"project-a", "project-b"} {
		roadyDir := filepath.Join(root, name, ".roady")
		if err := os.MkdirAll(roadyDir, 0700); err != nil {
			t.Fatal(err)
		}
	}

	svc := NewOrgService(root)
	projects, err := svc.DiscoverProjects()
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestOrgService_AggregateMetrics_Empty(t *testing.T) {
	root := t.TempDir()
	svc := NewOrgService(root)

	metrics, err := svc.AggregateMetrics()
	if err != nil {
		t.Fatal(err)
	}

	if metrics.TotalProjects != 0 {
		t.Errorf("expected 0 projects, got %d", metrics.TotalProjects)
	}
}

func TestOrgService_SaveAndLoadConfig(t *testing.T) {
	root := t.TempDir()
	svc := NewOrgService(root)

	config := &org.OrgConfig{
		Name:  "test-org",
		Repos: []string{"repo-a", "repo-b"},
	}

	if err := svc.SaveOrgConfig(config); err != nil {
		t.Fatal(err)
	}

	loaded, err := svc.LoadOrgConfig()
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Name != "test-org" {
		t.Errorf("expected name test-org, got %s", loaded.Name)
	}
	if len(loaded.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(loaded.Repos))
	}
}
