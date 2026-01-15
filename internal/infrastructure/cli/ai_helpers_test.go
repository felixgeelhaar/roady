package cli

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/spf13/cobra"
)

func TestPromptHelpers(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("custom\nn\n"))
	result, err := promptString(reader, "label", "default")
	if err != nil {
		t.Fatalf("promptString failed: %v", err)
	}
	if result != "custom" {
		t.Fatalf("expected custom, got %q", result)
	}

	reader = bufio.NewReader(strings.NewReader("y\nno\n"))
	got, err := promptBool(reader, "prompt", false)
	if err != nil {
		t.Fatalf("promptBool failed: %v", err)
	}
	if !got {
		t.Fatalf("expected true from 'y'")
	}

	reader = bufio.NewReader(strings.NewReader("\n"))
	defaultValue, _ := promptInt(reader, "number", 42)
	if defaultValue != 42 {
		t.Fatalf("expected default int 42, got %d", defaultValue)
	}

	reader = bufio.NewReader(strings.NewReader("13\n"))
	specified, err := promptInt(reader, "number", 0)
	if err != nil {
		t.Fatalf("promptInt failed: %v", err)
	}
	if specified != 13 {
		t.Fatalf("expected 13, got %d", specified)
	}

	if def := defaultString("", "fallback"); def != "fallback" {
		t.Fatalf("defaultString failed, got %q", def)
	}
}

func TestApplyAIFlags(t *testing.T) {
	prevProvider, prevModel := aiProvider, aiModel
	prevAllow, prevTokenLimit := aiAllow, aiTokenLimit
	defer func() {
		aiProvider = prevProvider
		aiModel = prevModel
		aiAllow = prevAllow
		aiTokenLimit = prevTokenLimit
	}()

	cmd := &cobra.Command{}
	cmd.Flags().Bool("allow-ai", false, "")
	cmd.Flags().Int("token-limit", 0, "")
	_ = cmd.Flags().Set("allow-ai", "true")
	_ = cmd.Flags().Set("token-limit", "123")

	aiAllow = true
	aiProvider = "openai"
	aiModel = "gpt-test"
	aiTokenLimit = 123

	cfg := &domain.PolicyConfig{}
	aiCfg := &config.AIConfig{}

	if err := applyAIFlags(cmd, cfg, aiCfg); err != nil {
		t.Fatalf("applyAIFlags failed: %v", err)
	}
	if !cfg.AllowAI || cfg.TokenLimit != 123 {
		t.Fatalf("policy config not updated: %+v", cfg)
	}
	if aiCfg.Provider != "openai" || aiCfg.Model != "gpt-test" {
		t.Fatalf("ai config not updated: %+v", aiCfg)
	}

	aiAllow = false
	aiProvider = ""
	aiModel = ""
	aiTokenLimit = 0

	cmd2 := &cobra.Command{}
	cmd2.Flags().Bool("allow-ai", false, "")
	cmd2.Flags().Int("token-limit", 0, "")
	if err := applyAIFlags(cmd2, cfg, aiCfg); err == nil {
		t.Fatal("expected error when no flags set")
	}
}

func TestValidateAIConfig(t *testing.T) {
	cfg := &domain.PolicyConfig{AllowAI: true}
	aiCfg := &config.AIConfig{Provider: "unsupported", Model: ""}
	if err := validateAIConfig(cfg, aiCfg); err == nil {
		t.Fatal("expected error for unsupported provider")
	}

	aiCfg.Provider = "mock"
	if err := validateAIConfig(cfg, aiCfg); err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestPromptAIConfigInteractive(t *testing.T) {
	tmp, err := os.CreateTemp("", "stdin-*")
	if err != nil {
		t.Fatalf("create stdin temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString("y\nmock\nmock-model\n77\n"); err != nil {
		t.Fatalf("write temp stdin: %v", err)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		t.Fatalf("seek temp stdin: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = tmp
	t.Cleanup(func() {
		os.Stdin = oldStdin
		tmp.Close()
	})

	cfg := &domain.PolicyConfig{}
	aiCfg := &config.AIConfig{}
	if err := promptAIConfig(cfg, aiCfg); err != nil {
		t.Fatalf("promptAIConfig failed: %v", err)
	}

	if !cfg.AllowAI || cfg.TokenLimit != 77 {
		t.Fatalf("policy config not updated: %+v", cfg)
	}
	if aiCfg.Provider != "mock" || aiCfg.Model != "mock-model" {
		t.Fatalf("ai config not updated: %+v", aiCfg)
	}
}
