package storage

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain/team"
	"gopkg.in/yaml.v3"
)

const TeamFile = "team.yaml"

func (r *FilesystemRepository) LoadTeam() (*team.TeamConfig, error) {
	path, err := r.ResolvePath(TeamFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &team.TeamConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read team file: %w", err)
	}

	var cfg team.TeamConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal team config: %w", err)
	}

	return &cfg, nil
}

func (r *FilesystemRepository) SaveTeam(cfg *team.TeamConfig) error {
	path, err := r.ResolvePath(TeamFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal team config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
