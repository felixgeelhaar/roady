package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
)

// DeadLetterStore appends failed webhook deliveries to a JSONL file.
type DeadLetterStore struct {
	path string
	mu   sync.Mutex
}

// NewDeadLetterStore creates a dead letter store at the given path.
func NewDeadLetterStore(path string) *DeadLetterStore {
	return &DeadLetterStore{path: path}
}

// Append writes a dead letter entry to the JSONL file.
func (s *DeadLetterStore) Append(dl events.DeadLetter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(dl)
	if err != nil {
		return fmt.Errorf("marshal dead letter: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open dead letter file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// ReadAll returns all dead letter entries from the file.
func (s *DeadLetterStore) ReadAll() ([]events.DeadLetter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []events.DeadLetter
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var dl events.DeadLetter
		if err := dec.Decode(&dl); err != nil {
			continue
		}
		entries = append(entries, dl)
	}
	return entries, nil
}
