package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAIConfigMissing(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir .roady: %v", err)
	}

	cfg, err := LoadAIConfig(tempDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config for missing file")
	}
}

func TestSaveAndLoadAIConfig(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir .roady: %v", err)
	}

	input := &AIConfig{Provider: "mock", Model: "test-model"}
	if err := SaveAIConfig(tempDir, input); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cfg, err := LoadAIConfig(tempDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected config")
	}
	if cfg.Provider != input.Provider || cfg.Model != input.Model {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadAIConfigInvalid(t *testing.T) {
	tempDir := t.TempDir()
	roadDir := filepath.Join(tempDir, ".roady")
	if err := os.MkdirAll(roadDir, 0700); err != nil {
		t.Fatalf("mkdir .roady: %v", err)
	}

	badPath := filepath.Join(roadDir, "ai.yaml")
	if err := os.WriteFile(badPath, []byte("::bad"), 0600); err != nil {
		t.Fatalf("write bad config: %v", err)
	}

	_, err := LoadAIConfig(tempDir)
	if err == nil {
		t.Fatalf("expected error for invalid yaml")
	}
}
