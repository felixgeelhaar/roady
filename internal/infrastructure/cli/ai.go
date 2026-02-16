package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	aiProvider    string
	aiModel       string
	aiAllow       bool
	aiTokenLimit  int
	aiInteractive bool
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Manage AI configuration",
}

var aiConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure AI provider and policy settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, cErr := getProjectRoot()
		if cErr != nil {
			return fmt.Errorf("resolve project path: %w", cErr)
		}
		repo := storage.NewFilesystemRepository(cwd)
		if !repo.IsInitialized() {
			return fmt.Errorf("roady is not initialized in this directory")
		}

		cfg, err := repo.LoadPolicy()
		if err != nil {
			return fmt.Errorf("failed to load policy: %w", err)
		}
		if cfg == nil {
			cfg = &domain.PolicyConfig{}
		}

		aiCfg, err := config.LoadAIConfig(cwd)
		if err != nil {
			return fmt.Errorf("failed to load AI config: %w", err)
		}
		if aiCfg == nil {
			aiCfg = &config.AIConfig{}
		}

		if aiInteractive {
			if err := promptAIConfig(cfg, aiCfg); err != nil {
				return err
			}
		} else {
			if err := applyAIFlags(cmd, cfg, aiCfg); err != nil {
				return err
			}
		}

		if err := validateAIConfig(cfg, aiCfg); err != nil {
			return err
		}

		if err := repo.SavePolicy(cfg); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}

		if err := config.SaveAIConfig(cwd, aiCfg); err != nil {
			return fmt.Errorf("failed to save AI config: %w", err)
		}

		fmt.Println("AI configuration saved.")
		fmt.Println("- Governance flags (allow_ai, token_limit) are stored in .roady/policy.yaml")
		fmt.Println("- Provider/model defaults live in .roady/ai.yaml and drive both CLI and MCP")
		return nil
	},
}

func runAIConfigureInteractive(repo *storage.FilesystemRepository) error {
	cfg, err := repo.LoadPolicy()
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}
	if cfg == nil {
		cfg = &domain.PolicyConfig{}
	}

	cwd, cErr := getProjectRoot()
	if cErr != nil {
		return fmt.Errorf("resolve project path: %w", cErr)
	}
	aiCfg, err := config.LoadAIConfig(cwd)
	if err != nil {
		return fmt.Errorf("failed to load AI config: %w", err)
	}
	if aiCfg == nil {
		aiCfg = &config.AIConfig{}
	}

	if err := promptAIConfig(cfg, aiCfg); err != nil {
		return err
	}

	if err := validateAIConfig(cfg, aiCfg); err != nil {
		return err
	}

	if err := repo.SavePolicy(cfg); err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	if err := config.SaveAIConfig(cwd, aiCfg); err != nil {
		return fmt.Errorf("failed to save AI config: %w", err)
	}

	fmt.Println("AI configuration saved.")
	fmt.Println("- Governance flags (allow_ai, token_limit) are stored in .roady/policy.yaml")
	fmt.Println("- Provider/model defaults live in .roady/ai.yaml and drive both CLI and MCP")
	return nil
}

func promptAIConfig(cfg *domain.PolicyConfig, aiCfg *config.AIConfig) error {
	reader := bufio.NewReader(os.Stdin)

	allow, err := promptBool(reader, "Enable AI?", cfg.AllowAI)
	if err != nil {
		return err
	}
	cfg.AllowAI = allow

	if cfg.AllowAI {
		provider, err := promptString(reader, "AI provider (ollama/openai/anthropic/gemini/mock)", defaultString(aiCfg.Provider, "ollama"))
		if err != nil {
			return err
		}
		aiCfg.Provider = provider

		model, err := promptString(reader, "AI model", defaultString(aiCfg.Model, "llama3"))
		if err != nil {
			return err
		}
		aiCfg.Model = model
	}

	limit, err := promptInt(reader, "Token limit (0 = unlimited)", cfg.TokenLimit)
	if err != nil {
		return err
	}
	cfg.TokenLimit = limit

	return nil
}

func applyAIFlags(cmd *cobra.Command, cfg *domain.PolicyConfig, aiCfg *config.AIConfig) error {
	if aiProvider == "" && aiModel == "" &&
		!cmd.Flags().Changed("allow-ai") && !cmd.Flags().Changed("token-limit") {
		return fmt.Errorf("no configuration provided; use flags or --interactive")
	}

	if cmd.Flags().Changed("allow-ai") {
		cfg.AllowAI = aiAllow
	}
	if aiProvider != "" {
		aiCfg.Provider = aiProvider
	}
	if aiModel != "" {
		aiCfg.Model = aiModel
	}
	if cmd.Flags().Changed("token-limit") {
		cfg.TokenLimit = aiTokenLimit
	}

	return nil
}

func validateAIConfig(cfg *domain.PolicyConfig, aiCfg *config.AIConfig) error {
	if !cfg.AllowAI {
		return nil
	}

	provider := strings.ToLower(strings.TrimSpace(aiCfg.Provider))
	switch provider {
	case "ollama", "openai", "anthropic", "gemini", "mock":
		// ok
	default:
		return fmt.Errorf("unsupported AI provider: %s", aiCfg.Provider)
	}

	if strings.TrimSpace(aiCfg.Model) == "" {
		return fmt.Errorf("AI model is required when AI is enabled")
	}

	return nil
}

func promptString(reader *bufio.Reader, label string, def string) (string, error) {
	fmt.Printf("%s [%s]: ", label, def)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(line)
	if value == "" {
		return def, nil
	}
	return value, nil
}

func promptBool(reader *bufio.Reader, label string, def bool) (bool, error) {
	defLabel := "n"
	if def {
		defLabel = "y"
	}
	for {
		fmt.Printf("%s [y/n] (%s): ", label, defLabel)
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		value := strings.ToLower(strings.TrimSpace(line))
		if value == "" {
			return def, nil
		}
		if value == "y" || value == "yes" {
			return true, nil
		}
		if value == "n" || value == "no" {
			return false, nil
		}
		fmt.Println("Please enter 'y' or 'n'.")
	}
}

func promptInt(reader *bufio.Reader, label string, def int) (int, error) {
	for {
		fmt.Printf("%s [%d]: ", label, def)
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		value := strings.TrimSpace(line)
		if value == "" {
			return def, nil
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			fmt.Println("Please enter a valid number.")
			continue
		}
		return parsed, nil
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func init() {
	aiConfigureCmd.Flags().StringVar(&aiProvider, "provider", "", "AI provider (ollama/openai/anthropic/gemini/mock)")
	aiConfigureCmd.Flags().StringVar(&aiModel, "model", "", "AI model identifier")
	aiConfigureCmd.Flags().BoolVar(&aiAllow, "allow-ai", false, "Enable AI usage")
	aiConfigureCmd.Flags().IntVar(&aiTokenLimit, "token-limit", 0, "Token limit (0 = unlimited)")
	aiConfigureCmd.Flags().BoolVar(&aiInteractive, "interactive", false, "Prompt for AI configuration interactively")

	aiCmd.AddCommand(aiConfigureCmd)
	RootCmd.AddCommand(aiCmd)
}
