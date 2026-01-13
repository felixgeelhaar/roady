package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain"
)

func (r *FilesystemRepository) RecordEvent(event domain.Event) error {
	path, err := r.ResolvePath(EventsFile)
	if err != nil {
		return err
	}
	
data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
data = append(data, '\n')

	// #nosec G304 -- Path is resolved and validated via resolvePath
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open events file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

func (r *FilesystemRepository) LoadEvents() ([]domain.Event, error) {
	path, err := r.ResolvePath(EventsFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via resolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Event{}, nil
		}
		return nil, fmt.Errorf("failed to read events file: %w", err)
	}

	var events []domain.Event
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var e domain.Event
		if err := json.Unmarshal(line, &e); err != nil {
			continue // Skip malformed lines
		}
		events = append(events, e)
	}

	return events, nil
}
