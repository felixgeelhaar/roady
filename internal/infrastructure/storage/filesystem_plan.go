package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

func (r *FilesystemRepository) SavePlan(p *planning.Plan) error {
	path, err := r.ResolvePath(PlanFile)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func (r *FilesystemRepository) LoadPlan() (*planning.Plan, error) {
	if _, err := os.Stat(r.root); err != nil {
		return nil, fmt.Errorf("root directory does not exist: %w", err)
	}

	path, err := r.ResolvePath(PlanFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via resolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return nil, nil to indicate no plan exists yet
		}
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var p planning.Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &p, nil
}