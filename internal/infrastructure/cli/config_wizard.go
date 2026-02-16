package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage project configuration",
}

var configWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Interactive configuration wizard for ai.yaml and policy.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, cErr := getProjectRoot()
		if cErr != nil {
			return fmt.Errorf("resolve project path: %w", cErr)
		}
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo

		reader := bufio.NewReader(os.Stdin)

		fmt.Println("\n--- Roady Configuration Wizard ---")
		fmt.Println("This wizard helps you configure AI provider and project policy settings.")
		fmt.Println()

		// --- AI Configuration ---
		fmt.Println("=== AI Provider Configuration (ai.yaml) ===")

		existing, _ := config.LoadAIConfig(cwd)
		if existing == nil {
			existing = &config.AIConfig{}
		}

		provider := prompt(reader, "AI Provider (ollama, openai, anthropic, gemini)", existing.Provider)
		model := prompt(reader, "Model name", existing.Model)
		maxRetriesStr := prompt(reader, "Max retries", intOrDefault(existing.MaxRetries, "2"))
		retryDelayStr := prompt(reader, "Retry delay (ms)", intOrDefault(existing.RetryDelayMs, "1000"))
		timeoutStr := prompt(reader, "Timeout (sec)", intOrDefault(existing.TimeoutSec, "300"))

		maxRetries, _ := strconv.Atoi(maxRetriesStr)
		retryDelay, _ := strconv.Atoi(retryDelayStr)
		timeout, _ := strconv.Atoi(timeoutStr)

		aiCfg := &config.AIConfig{
			Provider:     provider,
			Model:        model,
			MaxRetries:   maxRetries,
			RetryDelayMs: retryDelay,
			TimeoutSec:   timeout,
		}

		if err := config.SaveAIConfig(cwd, aiCfg); err != nil {
			return MapError(fmt.Errorf("failed to save AI config: %w", err))
		}
		fmt.Println("AI configuration saved.")
		fmt.Println()

		// --- Policy Configuration ---
		fmt.Println("=== Project Policy Configuration (policy.yaml) ===")

		existingPolicy, _ := repo.LoadPolicy()
		if existingPolicy == nil {
			existingPolicy = &domain.PolicyConfig{}
		}

		maxWIPStr := prompt(reader, "Max WIP (concurrent in-progress tasks)", intOrDefault(existingPolicy.MaxWIP, "3"))
		allowAIStr := prompt(reader, "Allow AI usage (true/false)", boolStr(existingPolicy.AllowAI))
		tokenLimitStr := prompt(reader, "AI token limit (0 = unlimited)", intOrDefault(existingPolicy.TokenLimit, "0"))

		maxWIP, _ := strconv.Atoi(maxWIPStr)
		allowAI := strings.ToLower(allowAIStr) == "true" || allowAIStr == "1" || allowAIStr == "yes"
		tokenLimit, _ := strconv.Atoi(tokenLimitStr)

		policyCfg := &domain.PolicyConfig{
			MaxWIP:     maxWIP,
			AllowAI:    allowAI,
			TokenLimit: tokenLimit,
		}

		if err := repo.SavePolicy(policyCfg); err != nil {
			return MapError(fmt.Errorf("failed to save policy: %w", err))
		}
		fmt.Println("Policy configuration saved.")

		fmt.Println("\n--- Configuration complete! ---")
		return nil
	},
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func intOrDefault(val int, fallback string) string {
	if val != 0 {
		return strconv.Itoa(val)
	}
	return fallback
}

func boolStr(val bool) string {
	if val {
		return "true"
	}
	return "false"
}

func init() {
	configCmd.AddCommand(configWizardCmd)
	RootCmd.AddCommand(configCmd)
}
