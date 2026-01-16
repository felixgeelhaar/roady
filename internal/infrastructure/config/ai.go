package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/roady/pkg/storage"
	"gopkg.in/yaml.v3"
)

const aiConfigFile = "ai.yaml"

// AIConfig stores provider defaults outside domain policy.
type AIConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`

	// Resilience settings
	MaxRetries   int `yaml:"max_retries"`    // Maximum retry attempts (default: 2)
	RetryDelayMs int `yaml:"retry_delay_ms"` // Initial retry delay in milliseconds (default: 1000)
	TimeoutSec   int `yaml:"timeout_sec"`    // Request timeout in seconds (default: 300)
}

func LoadAIConfig(root string) (*AIConfig, error) {
	repo := storage.NewFilesystemRepository(root)
	path, err := repo.ResolvePath(aiConfigFile)
	if err != nil {
		return nil, err
	}

	// Clean path for defense in depth (ResolvePath already sanitizes)
	cleanPath := filepath.Clean(path)
	// #nosec G304 -- path is resolved from trusted repository root, not user input
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read AI config: %w", err)
	}

	var cfg AIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI config: %w", err)
	}

	return &cfg, nil
}

func SaveAIConfig(root string, cfg *AIConfig) error {
	if cfg == nil {
		return fmt.Errorf("AI config is nil")
	}

	repo := storage.NewFilesystemRepository(root)
	path, err := repo.ResolvePath(aiConfigFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal AI config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
