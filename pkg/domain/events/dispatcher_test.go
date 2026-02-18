package events

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockEvent implements DomainEvent for testing
type mockEvent struct {
	eventType     string
	aggregateID   string
	aggregateType string
	timestamp     time.Time
	version       int
}

func (e mockEvent) EventType() string      { return e.eventType }
func (e mockEvent) AggregateID() string    { return e.aggregateID }
func (e mockEvent) AggregateType() string  { return e.aggregateType }
func (e mockEvent) OccurredAt() time.Time  { return e.timestamp }
func (e mockEvent) Version() int           { return e.version }

func newMockEvent(eventType string) mockEvent {
	return mockEvent{
		eventType:     eventType,
		aggregateID:   "test-aggregate",
		aggregateType: "test",
		timestamp:     time.Now(),
		version:       1,
	}
}

func TestEventDispatcher_Register(t *testing.T) {
	d := NewEventDispatcher()

	called := false
	d.RegisterHandler("test-handler", func(ctx context.Context, event DomainEvent) error {
		called = true
		return nil
	}, "test.event")

	if !d.HasHandlers("test.event") {
		t.Error("Expected handlers for test.event")
	}

	err := d.Dispatch(context.Background(), newMockEvent("test.event"))
	if err != nil {
		t.Errorf("Dispatch failed: %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestEventDispatcher_RegisterMultipleEventTypes(t *testing.T) {
	d := NewEventDispatcher()

	callCount := 0
	d.Register(HandlerRegistration{
		Name: "multi-handler",
		Handler: func(ctx context.Context, event DomainEvent) error {
			callCount++
			return nil
		},
		EventTypes: []string{"event.a", "event.b", "event.c"},
	})

	if err := d.Dispatch(context.Background(), newMockEvent("event.a")); err != nil { t.Fatal(err) }
	if err := d.Dispatch(context.Background(), newMockEvent("event.b")); err != nil { t.Fatal(err) }
	if err := d.Dispatch(context.Background(), newMockEvent("event.c")); err != nil { t.Fatal(err) }

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestEventDispatcher_Wildcard(t *testing.T) {
	d := NewEventDispatcher()

	callCount := 0
	d.RegisterWildcard("wildcard-handler", func(ctx context.Context, event DomainEvent) error {
		callCount++
		return nil
	})

	_ = d.Dispatch(context.Background(), newMockEvent("event.a"))
	_ = d.Dispatch(context.Background(), newMockEvent("event.b"))
	_ = d.Dispatch(context.Background(), newMockEvent("event.c"))

	if callCount != 3 {
		t.Errorf("Expected 3 calls for wildcard handler, got %d", callCount)
	}
}

func TestEventDispatcher_MultipleHandlers(t *testing.T) {
	d := NewEventDispatcher()

	handler1Called := false
	handler2Called := false

	d.RegisterHandler("handler1", func(ctx context.Context, event DomainEvent) error {
		handler1Called = true
		return nil
	}, "test.event")

	d.RegisterHandler("handler2", func(ctx context.Context, event DomainEvent) error {
		handler2Called = true
		return nil
	}, "test.event")

	_ = d.Dispatch(context.Background(), newMockEvent("test.event"))

	if !handler1Called {
		t.Error("Handler1 was not called")
	}
	if !handler2Called {
		t.Error("Handler2 was not called")
	}
}

func TestEventDispatcher_ErrorHandling(t *testing.T) {
	d := NewEventDispatcher()
	testErr := errors.New("handler error")

	d.RegisterHandler("failing-handler", func(ctx context.Context, event DomainEvent) error {
		return testErr
	}, "test.event")

	err := d.Dispatch(context.Background(), newMockEvent("test.event"))
	if err == nil {
		t.Error("Expected error from dispatch")
	}
}

func TestEventDispatcher_ContinueOnError(t *testing.T) {
	d := NewEventDispatcher()
	d.ContinueOnError = true

	handler1Called := false
	handler2Called := false

	d.RegisterHandler("failing-handler", func(ctx context.Context, event DomainEvent) error {
		handler1Called = true
		return errors.New("handler1 error")
	}, "test.event")

	d.RegisterHandler("succeeding-handler", func(ctx context.Context, event DomainEvent) error {
		handler2Called = true
		return nil
	}, "test.event")

	err := d.Dispatch(context.Background(), newMockEvent("test.event"))

	if !handler1Called {
		t.Error("Handler1 was not called")
	}
	if !handler2Called {
		t.Error("Handler2 should have been called despite handler1 error")
	}

	var dispatchErr *DispatchError
	if !errors.As(err, &dispatchErr) {
		t.Error("Expected DispatchError")
	}
	if len(dispatchErr.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(dispatchErr.Errors))
	}
}

func TestEventDispatcher_NoHandlers(t *testing.T) {
	d := NewEventDispatcher()

	err := d.Dispatch(context.Background(), newMockEvent("unhandled.event"))
	if err != nil {
		t.Errorf("Expected nil error for unhandled event, got: %v", err)
	}
}

func TestEventDispatcher_HandlerCount(t *testing.T) {
	d := NewEventDispatcher()

	if d.HandlerCount("test.event") != 0 {
		t.Error("Expected 0 handlers initially")
	}

	d.RegisterHandler("handler1", func(ctx context.Context, event DomainEvent) error {
		return nil
	}, "test.event")

	if d.HandlerCount("test.event") != 1 {
		t.Error("Expected 1 handler")
	}

	d.RegisterWildcard("wildcard", func(ctx context.Context, event DomainEvent) error {
		return nil
	})

	// Wildcard counts toward all event types
	if d.HandlerCount("test.event") != 2 {
		t.Error("Expected 2 handlers (1 specific + 1 wildcard)")
	}
	if d.HandlerCount("other.event") != 1 {
		t.Error("Expected 1 handler for other event (wildcard only)")
	}
}

func TestEventDispatcher_Clear(t *testing.T) {
	d := NewEventDispatcher()

	d.RegisterHandler("handler", func(ctx context.Context, event DomainEvent) error {
		return nil
	}, "test.event")

	if !d.HasHandlers("test.event") {
		t.Error("Expected handlers before clear")
	}

	d.Clear()

	if d.HasHandlers("test.event") {
		t.Error("Expected no handlers after clear")
	}
}

func TestEventDispatcher_DispatchAsync(t *testing.T) {
	d := NewEventDispatcher()

	called := make(chan bool, 1)
	d.RegisterHandler("async-handler", func(ctx context.Context, event DomainEvent) error {
		called <- true
		return nil
	}, "test.event")

	errChan := d.DispatchAsync(context.Background(), newMockEvent("test.event"))

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Async dispatch failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Async dispatch timed out")
	}

	select {
	case <-called:
		// Handler was called
	case <-time.After(time.Second):
		t.Error("Handler was not called")
	}
}

func TestDispatchError_Error(t *testing.T) {
	singleErr := &DispatchError{
		Errors: []error{errors.New("single error")},
	}
	if singleErr.Error() != "single error" {
		t.Errorf("Expected single error message, got: %s", singleErr.Error())
	}

	multiErr := &DispatchError{
		Errors: []error{
			errors.New("error 1"),
			errors.New("error 2"),
		},
	}
	expected := "multiple dispatch errors (2)"
	if multiErr.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, multiErr.Error())
	}
}

func TestDispatchError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	dispatchErr := &DispatchError{
		Errors: []error{originalErr},
	}

	if !errors.Is(dispatchErr, originalErr) {
		t.Error("Expected errors.Is to find original error")
	}

	emptyErr := &DispatchError{}
	if emptyErr.Unwrap() != nil {
		t.Error("Expected nil from empty DispatchError.Unwrap()")
	}
}
