package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanCodebaseTree(t *testing.T) {
	dir := t.TempDir()

	// Create some source files
	os.MkdirAll(filepath.Join(dir, "pkg", "domain"), 0755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "pkg", "domain", "model.go"), []byte("package domain"), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)

	// Create a hidden dir that should be skipped
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(""), 0644)

	result := ScanCodebaseTree(dir, 100)

	if !strings.Contains(result, "main.go") {
		t.Errorf("expected main.go in result, got: %s", result)
	}
	if !strings.Contains(result, filepath.Join("pkg", "domain", "model.go")) {
		t.Errorf("expected pkg/domain/model.go in result, got: %s", result)
	}
	if strings.Contains(result, ".git") {
		t.Errorf("should not contain .git, got: %s", result)
	}
}

func TestScanCodebaseTree_Truncation(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(dir, "file"+string(rune('a'+i))+".go"), []byte("package x"), 0644)
	}

	result := ScanCodebaseTree(dir, 3)
	if !strings.Contains(result, "truncated") {
		t.Errorf("expected truncation message, got: %s", result)
	}
}

func TestScanCodebaseTree_Empty(t *testing.T) {
	dir := t.TempDir()
	result := ScanCodebaseTree(dir, 100)
	if result != "(no source files found)" {
		t.Errorf("expected no files message, got: %s", result)
	}
}
