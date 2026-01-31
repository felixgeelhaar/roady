package storage

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain/messaging"
	"gopkg.in/yaml.v3"
)

const MessagingFile = "messaging.yaml"

// SaveMessagingConfig saves the messaging configuration.
func (r *FilesystemRepository) SaveMessagingConfig(config *messaging.MessagingConfig) error {
	path, err := r.ResolvePath(MessagingFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal messaging config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LoadMessagingConfig loads the messaging configuration.
func (r *FilesystemRepository) LoadMessagingConfig() (*messaging.MessagingConfig, error) {
	path, err := r.ResolvePath(MessagingFile)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &messaging.MessagingConfig{}, nil
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read messaging config: %w", err)
	}

	var config messaging.MessagingConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messaging config: %w", err)
	}

	return &config, nil
}
