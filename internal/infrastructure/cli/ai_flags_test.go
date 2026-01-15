package cli

import (
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/spf13/cobra"
)

func TestApplyAIFlagsRequiresFlags(t *testing.T) {
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

	cfg := &domain.PolicyConfig{}
	aiCfg := &config.AIConfig{}

	if err := applyAIFlags(cmd, cfg, aiCfg); err == nil {
		t.Fatal("expected error when no flags provided")
	}
}

func TestApplyAIFlagsSetsValues(t *testing.T) {
	prevProvider, prevModel := aiProvider, aiModel
	prevAllow, prevTokenLimit := aiAllow, aiTokenLimit
	defer func() {
		aiProvider = prevProvider
		aiModel = prevModel
		aiAllow = prevAllow
		aiTokenLimit = prevTokenLimit
	}()

	aiProvider = "openai"
	aiModel = "gpt-test"
	aiAllow = true
	aiTokenLimit = 150

	cmd := &cobra.Command{}
	cmd.Flags().Bool("allow-ai", false, "")
	cmd.Flags().Int("token-limit", 0, "")
	_ = cmd.Flags().Set("allow-ai", "true")
	_ = cmd.Flags().Set("token-limit", "150")

	cfg := &domain.PolicyConfig{}
	aiCfg := &config.AIConfig{}

	if err := applyAIFlags(cmd, cfg, aiCfg); err != nil {
		t.Fatalf("applyAIFlags failed: %v", err)
	}

	if !cfg.AllowAI || cfg.TokenLimit != 150 {
		t.Fatalf("policy not updated: %+v", cfg)
	}
	if aiCfg.Provider != "openai" || aiCfg.Model != "gpt-test" {
		t.Fatalf("ai config not updated: %+v", aiCfg)
	}
}

func TestValidateAIConfigDisallowsUnsupportedProvider(t *testing.T) {
	cfg := &domain.PolicyConfig{AllowAI: true}
	aiCfg := &config.AIConfig{Provider: "invalid", Model: "m"}

	if err := validateAIConfig(cfg, aiCfg); err == nil {
		t.Fatal("expected error for unsupported provider")
	}

	aiCfg.Provider = "mock"
	aiCfg.Model = ""
	if err := validateAIConfig(cfg, aiCfg); err == nil {
		t.Fatal("expected error when model missing")
	}
}

func TestValidateAIConfigSkipsWhenDisabled(t *testing.T) {
	cfg := &domain.PolicyConfig{AllowAI: false}
	aiCfg := &config.AIConfig{}

	if err := validateAIConfig(cfg, aiCfg); err != nil {
		t.Fatalf("expected no error when AI disabled, got %v", err)
	}
}
