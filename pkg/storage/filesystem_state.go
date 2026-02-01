package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func (r *FilesystemRepository) SaveState(s *planning.ExecutionState) error {
	path, err := r.ResolvePath(StateFile)
	if err != nil {
		return err
	}

	// Optimistic locking: read current version from disk and compare.
	// #nosec G304 -- Path is resolved and validated via ResolvePath
	existing, err := os.ReadFile(path)
	if err == nil {
		var disk planning.ExecutionState
		if jsonErr := json.Unmarshal(existing, &disk); jsonErr == nil {
			if disk.Version != s.Version {
				return &planning.ConflictError{Expected: s.Version, Actual: disk.Version}
			}
		}
	}
	// If file doesn't exist, no conflict possible.

	s.Version++

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

	// #nosec G304 -- Path is resolved and validated via ResolvePath
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
