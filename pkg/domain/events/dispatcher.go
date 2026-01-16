package events

import (
	"context"
	"fmt"
	"sync"
)

// EventHandlerFunc is a function that handles a domain event.
type EventHandlerFunc func(ctx context.Context, event DomainEvent) error

// HandlerRegistration represents a handler registration for specific event types.
type HandlerRegistration struct {
	EventTypes []string
	Handler    EventHandlerFunc
	Name       string // For logging/debugging
}

// EventDispatcher dispatches domain events to registered handlers.
type EventDispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]namedHandler
	// ContinueOnError determines if dispatch should continue when a handler fails
	ContinueOnError bool
}

// namedHandler wraps a handler with its name for debugging
type namedHandler struct {
	name    string
	handler EventHandlerFunc
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers:        make(map[string][]namedHandler),
		ContinueOnError: false,
	}
}

// Register registers a handler for specific event types.
func (d *EventDispatcher) Register(reg HandlerRegistration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	nh := namedHandler{
		name:    reg.Name,
		handler: reg.Handler,
	}

	for _, eventType := range reg.EventTypes {
		d.handlers[eventType] = append(d.handlers[eventType], nh)
	}
}

// RegisterHandler is a convenience method to register a single handler for event types.
func (d *EventDispatcher) RegisterHandler(name string, handler EventHandlerFunc, eventTypes ...string) {
	d.Register(HandlerRegistration{
		Name:       name,
		Handler:    handler,
		EventTypes: eventTypes,
	})
}

// RegisterWildcard registers a handler for all events (wildcard "*").
func (d *EventDispatcher) RegisterWildcard(name string, handler EventHandlerFunc) {
	d.RegisterHandler(name, handler, "*")
}

// Dispatch dispatches an event to all registered handlers.
// If ContinueOnError is false, dispatch stops at the first error.
// If ContinueOnError is true, all handlers are executed and errors are collected.
func (d *EventDispatcher) Dispatch(ctx context.Context, event DomainEvent) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	eventType := event.EventType()

	// Collect handlers for this specific event type and wildcard handlers
	var handlers []namedHandler
	handlers = append(handlers, d.handlers[eventType]...)
	handlers = append(handlers, d.handlers["*"]...)

	if len(handlers) == 0 {
		return nil
	}

	var errors []error
	for _, nh := range handlers {
		if err := nh.handler(ctx, event); err != nil {
			handlerErr := fmt.Errorf("handler %s failed for event %s: %w", nh.name, eventType, err)
			if !d.ContinueOnError {
				return handlerErr
			}
			errors = append(errors, handlerErr)
		}
	}

	if len(errors) > 0 {
		return &DispatchError{Errors: errors}
	}

	return nil
}

// DispatchAsync dispatches an event asynchronously.
// Returns a channel that receives the error (or nil) when dispatch completes.
func (d *EventDispatcher) DispatchAsync(ctx context.Context, event DomainEvent) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Dispatch(ctx, event)
		close(errChan)
	}()
	return errChan
}

// HasHandlers returns true if there are handlers registered for the given event type.
func (d *EventDispatcher) HasHandlers(eventType string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.handlers[eventType]) > 0 || len(d.handlers["*"]) > 0
}

// HandlerCount returns the number of handlers registered for a specific event type.
func (d *EventDispatcher) HandlerCount(eventType string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	count := len(d.handlers[eventType])
	if eventType != "*" {
		count += len(d.handlers["*"])
	}
	return count
}

// Clear removes all registered handlers.
func (d *EventDispatcher) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers = make(map[string][]namedHandler)
}

// DispatchError contains multiple errors from event dispatch.
type DispatchError struct {
	Errors []error
}

func (e *DispatchError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("multiple dispatch errors (%d)", len(e.Errors))
}

// Unwrap returns the first error for errors.Is/As support.
func (e *DispatchError) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}
