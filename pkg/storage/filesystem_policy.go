package storage

import (
	"bytes"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"gopkg.in/yaml.v3"
)

func (r *FilesystemRepository) LoadPolicy() (*domain.PolicyConfig, error) {
	path, err := r.ResolvePath(PolicyFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &domain.PolicyConfig{MaxWIP: 3, AllowAI: true}, nil // Default
		}
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var cfg domain.PolicyConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err == nil {
		return &cfg, nil
	}

	// Legacy policy support for deprecated provider/model fields.
	type legacyPolicyConfig struct {
		MaxWIP     int    `yaml:"max_wip"`
		AllowAI    bool   `yaml:"allow_ai"`
		TokenLimit int    `yaml:"token_limit"`
		AIProvider string `yaml:"ai_provider"`
		AIModel    string `yaml:"ai_model"`
	}

	var legacy legacyPolicyConfig
	decLegacy := yaml.NewDecoder(bytes.NewReader(data))
	decLegacy.KnownFields(true)
	if err := decLegacy.Decode(&legacy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	return &domain.PolicyConfig{
		MaxWIP:     legacy.MaxWIP,
		AllowAI:    legacy.AllowAI,
		TokenLimit: legacy.TokenLimit,
	}, nil
}

func (r *FilesystemRepository) SavePolicy(cfg *domain.PolicyConfig) error {
	path, err := r.ResolvePath(PolicyFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
