package application_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
)

func TestDetectCrossDrift_NoProjects(t *testing.T) {
	root := t.TempDir()

	svc := application.NewOrgService(root)
	report, err := svc.DetectCrossDrift()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(report.Projects))
	}
	if report.TotalIssues != 0 {
		t.Errorf("expected 0 total issues, got %d", report.TotalIssues)
	}
}

func TestDetectCrossDrift_ProjectsFound(t *testing.T) {
	root := t.TempDir()

	// Create two project dirs with .roady
	for _, name := range []string{"proj-a", "proj-b"} {
		dir := filepath.Join(root, name, ".roady")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}

	svc := application.NewOrgService(root)
	report, err := svc.DetectCrossDrift()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(report.Projects))
	}
}
