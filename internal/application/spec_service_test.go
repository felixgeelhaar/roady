package application_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/domain/spec"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
)

func TestSpecService_ImportFromMarkdown(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-spec-test-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	service := application.NewSpecService(repo)

	mdPath := filepath.Join(tempDir, "test.md")
	content := "# My Project\n\n## Feature 1\nDescription 1"
	os.WriteFile(mdPath, []byte(content), 0600)

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
	defer os.RemoveAll(tempDir)
	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	service := application.NewSpecService(repo)

	mdPath := filepath.Join(tempDir, "complex.md")
	content := "# Project\nHigh level desc\n\n## F1\nDesc 1\n- item 1\n\n## F2\nDesc 2"
	os.WriteFile(mdPath, []byte(content), 0600)

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
	defer os.RemoveAll(tempDir)
	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	service := application.NewSpecService(repo)

	mdPath := filepath.Join(tempDir, "leading.md")
	content := "This is a leading description.\n\n# Project Name"
	os.WriteFile(mdPath, []byte(content), 0600)

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
	defer os.Remove(tempFile.Name())
	tempFile.WriteString("# Hello")
	tempFile.Close()

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
	