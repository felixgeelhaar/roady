package application_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestSpecService_ImportFromMarkdown(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-spec-test-*")
	defer func() { _ = os.RemoveAll(tempDir) }()

	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	service := application.NewSpecService(repo)

	mdPath := filepath.Join(tempDir, "test.md")
	content := "# My Project\n\n## Feature 1\nDescription 1"
	_ = os.WriteFile(mdPath, []byte(content), 0600)

	s, err := service.ImportFromMarkdown(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "My Project" {
		t.Errorf("Expected title My Project, got %s", s.Title)
	}
}

func TestSpecService_ComplexMarkdown(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-spec-complex-*")
	defer func() { _ = os.RemoveAll(tempDir) }()
	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	service := application.NewSpecService(repo)

	mdPath := filepath.Join(tempDir, "complex.md")
	content := "# Project\nHigh level desc\n\n## F1\nDesc 1\n- item 1\n\n## F2\nDesc 2"
	_ = os.WriteFile(mdPath, []byte(content), 0600)

	s, err := service.ImportFromMarkdown(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(s.Features))
	}
}

func TestSpecService_LeadingDesc(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-spec-leading-*")
	defer func() { _ = os.RemoveAll(tempDir) }()
	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	service := application.NewSpecService(repo)

	mdPath := filepath.Join(tempDir, "leading.md")
	content := "This is a leading description.\n\n# Project Name"
	_ = os.WriteFile(mdPath, []byte(content), 0600)

	s, _ := service.ImportFromMarkdown(mdPath)
	if s.Description != "This is a leading description." {
		t.Errorf("expected leading desc, got %s", s.Description)
	}
}

func TestSpecService_GetSpec(t *testing.T) {
	repo := &MockRepo{Spec: &spec.ProductSpec{ID: "s1"}}
	service := application.NewSpecService(repo)
	s, _ := service.GetSpec()
	if s.ID != "s1" {
		t.Errorf("GetSpec failed")
	}
}

func TestSpecService_Import_Mock(t *testing.T) {
	repo := &MockRepo{}
	service := application.NewSpecService(repo)

	tempFile, _ := os.CreateTemp("", "import-*.md")
	defer func() { _ = os.Remove(tempFile.Name()) }()
	if _, err := tempFile.WriteString("# Hello"); err != nil {
		t.Fatal(err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := service.ImportFromMarkdown(tempFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if repo.Spec.Title != "Hello" {
		t.Errorf("Expected Title Hello, got %s", repo.Spec.Title)
	}
}

func TestSpecService_ImportError(t *testing.T) {
	repo := &MockRepo{}
	service := application.NewSpecService(repo)

	// File not found
	_, err := service.ImportFromMarkdown("/tmp/nonexistent-file-12345")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSpecService_AnalyzeDirectory(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-spec-analyze-*")
	defer func() { _ = os.RemoveAll(tempDir) }()
	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	service := application.NewSpecService(repo)

	first := filepath.Join(tempDir, "a.md")
	second := filepath.Join(tempDir, "b.md")
	_ = os.WriteFile(first, []byte("# Project\n\n## Feature One\nDesc A"), 0600)
	_ = os.WriteFile(second, []byte("# Project\n\n## Feature One\nDesc B"), 0600)

	spec, err := service.AnalyzeDirectory(tempDir)
	if err != nil {
		t.Fatalf("AnalyzeDirectory failed: %v", err)
	}
	if len(spec.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(spec.Features))
	}
	if spec.Title != "Project" {
		t.Fatalf("expected project title, got %q", spec.Title)
	}
	if !strings.Contains(spec.Features[0].Description, "Desc B") {
		t.Fatalf("expected merged description, got %q", spec.Features[0].Description)
	}
}

func TestSpecService_AddFeature(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-spec-add-*")
	defer func() { _ = os.RemoveAll(tempDir) }()
	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	service := application.NewSpecService(repo)

	if err := repo.SaveSpec(&spec.ProductSpec{
		ID:      "spec-1",
		Title:   "Project",
		Version: "0.1.0",
		Features: []spec.Feature{
			{ID: "f1", Title: "Feature 1"},
		},
	}); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	oldWD, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	updated, err := service.AddFeature("Feature 2", "Desc 2")
	if err != nil {
		t.Fatalf("AddFeature failed: %v", err)
	}
	if len(updated.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(updated.Features))
	}
	content, err := os.ReadFile(filepath.Join(tempDir, "docs", "backlog.md"))
	if err != nil {
		t.Fatalf("read backlog: %v", err)
	}
	if !strings.Contains(string(content), "Feature 2") {
		t.Fatalf("expected backlog to include feature, got %q", string(content))
	}
}
