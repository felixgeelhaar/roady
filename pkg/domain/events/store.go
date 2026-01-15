package events

import (
	"time"
)

// EventStore provides persistence for domain events.
type EventStore interface {
	// Append adds a new event to the store, chaining it to the previous event.
	Append(event *BaseEvent) error

	// LoadAll returns all events in chronological order.
	LoadAll() ([]*BaseEvent, error)

	// LoadByAggregate returns events for a specific aggregate.
	LoadByAggregate(aggregateType, aggregateID string) ([]*BaseEvent, error)

	// LoadByType returns events of a specific type.
	LoadByType(eventType string) ([]*BaseEvent, error)

	// LoadSince returns events that occurred after the given timestamp.
	LoadSince(since time.Time) ([]*BaseEvent, error)

	// LoadRange returns events within a time range.
	LoadRange(from, to time.Time) ([]*BaseEvent, error)

	// GetLastEvent returns the most recent event (for hash chaining).
	GetLastEvent() (*BaseEvent, error)

	// Count returns the total number of events.
	Count() (int, error)
}

// Projection rebuilds state from events.
type Projection interface {
	// Name returns the projection name for identification.
	Name() string

	// Apply processes a single event to update the projection state.
	Apply(event *BaseEvent) error

	// Rebuild reprocesses all events to rebuild the projection from scratch.
	Rebuild(events []*BaseEvent) error

	// Reset clears the projection state.
	Reset() error
}

// ProjectionStore persists projection state.
type ProjectionStore interface {
	// SaveCheckpoint saves the last processed event ID for a projection.
	SaveCheckpoint(projectionName string, lastEventID string) error

	// LoadCheckpoint returns the last processed event ID.
	LoadCheckpoint(projectionName string) (string, error)
}

// EventPublisher broadcasts events to subscribers.
type EventPublisher interface {
	// Publish sends an event to all registered subscribers.
	Publish(event *BaseEvent) error

	// Subscribe registers a handler for events.
	Subscribe(handler EventHandler)
}

// EventHandler processes published events.
type EventHandler func(event *BaseEvent) error

// EventQuery provides filtering options for event queries.
type EventQuery struct {
	AggregateType string
	AggregateID   string
	EventTypes    []string
	Since         *time.Time
	Until         *time.Time
	Limit         int
	Offset        int
}
