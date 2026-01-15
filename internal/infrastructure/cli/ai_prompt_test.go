package cli

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestPromptStringUsesDefaultWhenEmpty(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	value, err := promptString(reader, "label", "fallback")
	if err != nil {
		t.Fatalf("promptString: %v", err)
	}
	if value != "fallback" {
		t.Fatalf("expected fallback, got %q", value)
	}
}

func TestPromptStringReturnsTrimmedInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(" custom-value \n"))
	value, err := promptString(reader, "label", "fallback")
	if err != nil {
		t.Fatalf("promptString: %v", err)
	}
	if value != "custom-value" {
		t.Fatalf("expected trimmed input, got %q", value)
	}
}

func TestPromptBoolRetriesOnInvalid(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("maybe\ny\n"))
	value, err := promptBool(reader, "label", false)
	if err != nil {
		t.Fatalf("promptBool: %v", err)
	}
	if !value {
		t.Fatalf("expected true after retry")
	}
}

func TestPromptIntRejectsBadInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("nope\n123\n"))
	value, err := promptInt(reader, "label", 5)
	if err != nil {
		t.Fatalf("promptInt: %v", err)
	}
	if value != 123 {
		t.Fatalf("expected parsed value, got %d", value)
	}
}

func TestDefaultStringFallback(t *testing.T) {
	if defaultString("  ", "fallback") != "fallback" {
		t.Fatalf("expected fallback for empty string")
	}
	if defaultString("value", "fallback") != "value" {
		t.Fatalf("expected actual value")
	}
}

func TestRunAIConfigureInteractiveWritesConfig(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	tmpFile := stdinFromString(t, "y\nmock\ntest-model\n150\n")
	origStdin := os.Stdin
	os.Stdin = tmpFile
	t.Cleanup(func() {
		os.Stdin = origStdin
		tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	})

	if err := runAIConfigureInteractive(repo); err != nil {
		t.Fatalf("runAIConfigureInteractive: %v", err)
	}

	policy, err := repo.LoadPolicy()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if policy == nil || !policy.AllowAI || policy.TokenLimit != 150 {
		t.Fatalf("unexpected policy: %+v", policy)
	}

	aiCfg, err := config.LoadAIConfig(tempDir)
	if err != nil {
		t.Fatalf("load ai config: %v", err)
	}
	if aiCfg == nil || aiCfg.Provider != "mock" || aiCfg.Model != "test-model" {
		t.Fatalf("unexpected ai config: %+v", aiCfg)
	}
}

func TestRunAIConfigureInteractiveDisablesAI(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	tmpFile := stdinFromString(t, "n\n42\n")
	origStdin := os.Stdin
	os.Stdin = tmpFile
	t.Cleanup(func() {
		os.Stdin = origStdin
		tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	})

	if err := runAIConfigureInteractive(repo); err != nil {
		t.Fatalf("runAIConfigureInteractive: %v", err)
	}

	policy, err := repo.LoadPolicy()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if policy == nil || policy.AllowAI || policy.TokenLimit != 42 {
		t.Fatalf("unexpected policy: %+v", policy)
	}

	aiCfg, err := config.LoadAIConfig(tempDir)
	if err != nil {
		t.Fatalf("load ai config: %v", err)
	}
	if aiCfg == nil {
		t.Fatalf("expected ai config struct")
	}
	if aiCfg.Provider != "" || aiCfg.Model != "" {
		t.Fatalf("expected empty ai config, got %+v", aiCfg)
	}
}

func stdinFromString(t *testing.T, data string) *os.File {
	t.Helper()
	file, err := os.CreateTemp("", "roady-stdin")
	if err != nil {
		t.Fatalf("create stdin temp file: %v", err)
	}
	if _, err := file.WriteString(data); err != nil {
		t.Fatalf("write stdin data: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("seek stdin file: %v", err)
	}
	return file
}
