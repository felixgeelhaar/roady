package wiring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
)

func TestLoadAIProviderDefaults(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir .roady: %v", err)
	}

	prevProvider := os.Getenv("ROADY_AI_PROVIDER")
	prevModel := os.Getenv("ROADY_AI_MODEL")
	os.Unsetenv("ROADY_AI_PROVIDER")
	os.Unsetenv("ROADY_AI_MODEL")
	t.Cleanup(func() {
		if prevProvider == "" {
			os.Unsetenv("ROADY_AI_PROVIDER")
		} else {
			_ = os.Setenv("ROADY_AI_PROVIDER", prevProvider)
		}
		if prevModel == "" {
			os.Unsetenv("ROADY_AI_MODEL")
		} else {
			_ = os.Setenv("ROADY_AI_MODEL", prevModel)
		}
	})

	provider, err := LoadAIProvider(tempDir)
	if err != nil {
		t.Fatalf("load provider: %v", err)
	}
	if provider.ID() != "ollama:llama3" {
		t.Fatalf("unexpected provider id: %s", provider.ID())
	}
}

func TestLoadAIProviderFromConfig(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir .roady: %v", err)
	}

	cfg := &config.AIConfig{Provider: "mock", Model: "test"}
	if err := config.SaveAIConfig(tempDir, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	prevProvider := os.Getenv("ROADY_AI_PROVIDER")
	prevModel := os.Getenv("ROADY_AI_MODEL")
	os.Unsetenv("ROADY_AI_PROVIDER")
	os.Unsetenv("ROADY_AI_MODEL")
	t.Cleanup(func() {
		if prevProvider == "" {
			os.Unsetenv("ROADY_AI_PROVIDER")
		} else {
			_ = os.Setenv("ROADY_AI_PROVIDER", prevProvider)
		}
		if prevModel == "" {
			os.Unsetenv("ROADY_AI_MODEL")
		} else {
			_ = os.Setenv("ROADY_AI_MODEL", prevModel)
		}
	})

	provider, err := LoadAIProvider(tempDir)
	if err != nil {
		t.Fatalf("load provider: %v", err)
	}
	if provider.ID() != "mock:test" {
		t.Fatalf("unexpected provider id: %s", provider.ID())
	}
}
