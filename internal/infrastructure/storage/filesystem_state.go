package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

func (r *FilesystemRepository) SaveState(s *planning.ExecutionState) error {
	path, err := r.ResolvePath(StateFile)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func (r *FilesystemRepository) LoadState() (*planning.ExecutionState, error) {
	path, err := r.ResolvePath(StateFile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state if not found
			return planning.NewExecutionState("unknown"), nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var s planning.ExecutionState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &s, nil
}
