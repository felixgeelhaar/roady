package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestAIConfigureWritesConfig(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	cwd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	if err := aiConfigureCmd.Flags().Set("provider", "mock"); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if err := aiConfigureCmd.Flags().Set("model", "test-model"); err != nil {
		t.Fatalf("set model: %v", err)
	}
	if err := aiConfigureCmd.Flags().Set("allow-ai", "true"); err != nil {
		t.Fatalf("set allow-ai: %v", err)
	}
	if err := aiConfigureCmd.Flags().Set("token-limit", "123"); err != nil {
		t.Fatalf("set token-limit: %v", err)
	}

	if err := aiConfigureCmd.RunE(aiConfigureCmd, []string{}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	policy, err := repo.LoadPolicy()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if policy.AllowAI != true || policy.TokenLimit != 123 {
		t.Fatalf("unexpected policy: %+v", policy)
	}

	aiCfg, err := config.LoadAIConfig(tempDir)
	if err != nil {
		t.Fatalf("load ai config: %v", err)
	}
	if aiCfg == nil {
		t.Fatalf("expected ai config")
	}
	if aiCfg.Provider != "mock" || aiCfg.Model != "test-model" {
		t.Fatalf("unexpected ai config: %+v", aiCfg)
	}

	// Ensure file exists in .roady
	if _, err := os.Stat(filepath.Join(tempDir, ".roady", "ai.yaml")); err != nil {
		t.Fatalf("expected ai.yaml: %v", err)
	}
}
