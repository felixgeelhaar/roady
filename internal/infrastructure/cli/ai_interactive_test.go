package cli

import (
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestRunAIConfigureInteractive(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	tmp, err := os.CreateTemp("", "ai-config-stdin-*")
	if err != nil {
		t.Fatalf("create temp stdin: %v", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString("y\nmock\nmock-model\n42\n"); err != nil {
		t.Fatalf("write temp stdin: %v", err)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		t.Fatalf("seek temp stdin: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = tmp
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = tmp.Close()
	})

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	cfg := &domain.PolicyConfig{}
	aiCfg := &config.AIConfig{}

	if err := runAIConfigureInteractive(repo); err != nil {
		t.Fatalf("runAIConfigureInteractive failed: %v", err)
	}

	loadedPolicy, err := repo.LoadPolicy()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if !loadedPolicy.AllowAI || loadedPolicy.TokenLimit != 42 {
		t.Fatalf("policy not updated: %+v", loadedPolicy)
	}

	loadedAI, err := config.LoadAIConfig(tempDir)
	if err != nil {
		t.Fatalf("load ai config: %v", err)
	}
	if loadedAI.Provider != "mock" || loadedAI.Model != "mock-model" {
		t.Fatalf("ai config not updated: %+v", loadedAI)
	}

	if cfg.AllowAI || cfg.TokenLimit != 0 || aiCfg.Provider != "" {
		t.Fatalf("input configs should remain untouched")
	}
}
