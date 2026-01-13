package config

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/storage"
	"gopkg.in/yaml.v3"
)

const aiConfigFile = "ai.yaml"

// AIConfig stores provider defaults outside domain policy.
type AIConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

func LoadAIConfig(root string) (*AIConfig, error) {
	repo := storage.NewFilesystemRepository(root)
	path, err := repo.ResolvePath(aiConfigFile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
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
