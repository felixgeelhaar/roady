package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/google/uuid"
)

// FileEventStore implements EventStore using a JSON Lines file.
type FileEventStore struct {
	mu       sync.RWMutex
	path     string
	lastHash string
}

// NewFileEventStore creates a new file-based event store.
func NewFileEventStore(basePath string) (*FileEventStore, error) {
	path := filepath.Join(basePath, "events.jsonl")

	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	store := &FileEventStore{path: path}

	// Load last hash for chaining
	if last, err := store.GetLastEvent(); err == nil && last != nil {
		store.lastHash = last.Hash
	}

	return store, nil
}

// Append adds a new event to the store.
func (s *FileEventStore) Append(event *events.BaseEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set ID if not provided
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Chain to previous event
	event.PrevHash = s.lastHash
	event.Hash = event.CalculateHash()

	// Open file in append mode
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open events file: %w", err)
	}
	defer f.Close()

	// Write JSON line
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	s.lastHash = event.Hash
	return nil
}

// LoadAll returns all events in chronological order.
func (s *FileEventStore) LoadAll() ([]*events.BaseEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadEvents()
}

// LoadByAggregate returns events for a specific aggregate.
func (s *FileEventStore) LoadByAggregate(aggregateType, aggregateID string) ([]*events.BaseEvent, error) {
	all, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	var result []*events.BaseEvent
	for _, e := range all {
		if e.AggregateType_ == aggregateType && e.AggregateID_ == aggregateID {
			result = append(result, e)
		}
	}
	return result, nil
}

// LoadByType returns events of a specific type.
func (s *FileEventStore) LoadByType(eventType string) ([]*events.BaseEvent, error) {
	all, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	var result []*events.BaseEvent
	for _, e := range all {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result, nil
}

// LoadSince returns events that occurred after the given timestamp.
func (s *FileEventStore) LoadSince(since time.Time) ([]*events.BaseEvent, error) {
	all, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	var result []*events.BaseEvent
	for _, e := range all {
		if e.Timestamp.After(since) {
			result = append(result, e)
		}
	}
	return result, nil
}

// LoadRange returns events within a time range.
func (s *FileEventStore) LoadRange(from, to time.Time) ([]*events.BaseEvent, error) {
	all, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	var result []*events.BaseEvent
	for _, e := range all {
		if (e.Timestamp.Equal(from) || e.Timestamp.After(from)) &&
			(e.Timestamp.Equal(to) || e.Timestamp.Before(to)) {
			result = append(result, e)
		}
	}
	return result, nil
}

// GetLastEvent returns the most recent event.
func (s *FileEventStore) GetLastEvent() (*events.BaseEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	evts, err := s.loadEvents()
	if err != nil {
		return nil, err
	}

	if len(evts) == 0 {
		return nil, nil
	}

	return evts[len(evts)-1], nil
}

// Count returns the total number of events.
func (s *FileEventStore) Count() (int, error) {
	evts, err := s.LoadAll()
	if err != nil {
		return 0, err
	}
	return len(evts), nil
}

// VerifyIntegrity checks the hash chain for tampering.
func (s *FileEventStore) VerifyIntegrity() ([]string, error) {
	evts, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	var violations []string
	lastHash := ""

	for i, e := range evts {
		// Verify chain
		if e.PrevHash != lastHash {
			violations = append(violations, fmt.Sprintf("Event %d (%s): PrevHash mismatch", i, e.ID))
		}

		// Verify self-hash
		expected := e.CalculateHash()
		if e.Hash != expected {
			violations = append(violations, fmt.Sprintf("Event %d (%s): Hash mismatch - possible tampering", i, e.ID))
		}

		lastHash = e.Hash
	}

	return violations, nil
}

// loadEvents reads all events from the file.
func (s *FileEventStore) loadEvents() ([]*events.BaseEvent, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open events file: %w", err)
	}
	defer f.Close()

	var result []*events.BaseEvent
	scanner := bufio.NewScanner(f)

	// Increase buffer size for large events
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event events.BaseEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("unmarshal event: %w", err)
		}
		result = append(result, &event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan events: %w", err)
	}

	return result, nil
}

// InMemoryEventPublisher is a simple in-process event publisher.
type InMemoryEventPublisher struct {
	mu       sync.RWMutex
	handlers []events.EventHandler
}

// NewInMemoryEventPublisher creates a new in-memory publisher.
func NewInMemoryEventPublisher() *InMemoryEventPublisher {
	return &InMemoryEventPublisher{
		handlers: make([]events.EventHandler, 0),
	}
}

// Publish sends an event to all subscribers.
func (p *InMemoryEventPublisher) Publish(event *events.BaseEvent) error {
	p.mu.RLock()
	handlers := make([]events.EventHandler, len(p.handlers))
	copy(handlers, p.handlers)
	p.mu.RUnlock()

	for _, h := range handlers {
		if err := h(event); err != nil {
			// Log error but don't fail - handlers shouldn't block publishing
			continue
		}
	}
	return nil
}

// Subscribe registers a handler for events.
func (p *InMemoryEventPublisher) Subscribe(handler events.EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers = append(p.handlers, handler)
}
