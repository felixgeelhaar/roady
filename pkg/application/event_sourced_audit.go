// Package application provides application services.
package application

import (
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/google/uuid"
)

// EventSourcedAuditService implements AuditLogger using the event store.
// It bridges the existing audit interface with the new event sourcing system.
type EventSourcedAuditService struct {
	store      events.EventStore
	publisher  events.EventPublisher
	taskProj   *events.TaskStateProjection
	velProj    *events.VelocityProjection
	auditProj  *events.AuditTimelineProjection
}

// Compile-time check that EventSourcedAuditService implements AuditLogger.
var _ domain.AuditLogger = (*EventSourcedAuditService)(nil)

// NewEventSourcedAuditService creates a new event-sourced audit service.
func NewEventSourcedAuditService(store events.EventStore, publisher events.EventPublisher) (*EventSourcedAuditService, error) {
	svc := &EventSourcedAuditService{
		store:     store,
		publisher: publisher,
		taskProj:  events.NewTaskStateProjection(),
		velProj:   events.NewVelocityProjection(7),
		auditProj: events.NewAuditTimelineProjection(),
	}

	// Rebuild projections from existing events
	if err := svc.rebuildProjections(); err != nil {
		return nil, err
	}

	// Subscribe projections to new events
	if publisher != nil {
		publisher.Subscribe(func(e *events.BaseEvent) error {
			svc.taskProj.Apply(e)
			svc.velProj.Apply(e)
			svc.auditProj.Apply(e)
			return nil
		})
	}

	return svc, nil
}

func (s *EventSourcedAuditService) rebuildProjections() error {
	evts, err := s.store.LoadAll()
	if err != nil {
		return err
	}

	if err := s.taskProj.Rebuild(evts); err != nil {
		return err
	}
	if err := s.velProj.Rebuild(evts); err != nil {
		return err
	}
	if err := s.auditProj.Rebuild(evts); err != nil {
		return err
	}

	return nil
}

// Log implements domain.AuditLogger.
func (s *EventSourcedAuditService) Log(action string, actor string, metadata map[string]interface{}) error {
	event := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      action,
		Timestamp: time.Now(),
		Actor:     actor,
		Metadata:  metadata,
	}

	// Extract aggregate info from metadata if available
	if taskID, ok := metadata["task_id"].(string); ok {
		event.AggregateID_ = taskID
		event.AggregateType_ = events.AggregateTypeTask
	} else if planID, ok := metadata["plan_id"].(string); ok {
		event.AggregateID_ = planID
		event.AggregateType_ = events.AggregateTypePlan
	}

	if err := s.store.Append(event); err != nil {
		return err
	}

	// Publish to subscribers (projections)
	if s.publisher != nil {
		s.publisher.Publish(event)
	}

	return nil
}

// GetTimeline returns the audit timeline from the projection.
func (s *EventSourcedAuditService) GetTimeline() []events.TimelineEntry {
	return s.auditProj.GetTimeline()
}

// GetRecentTimeline returns the most recent n timeline entries.
func (s *EventSourcedAuditService) GetRecentTimeline(n int) []events.TimelineEntry {
	return s.auditProj.GetRecentEntries(n)
}

// GetTaskState returns the current state of a task from the projection.
func (s *EventSourcedAuditService) GetTaskState(taskID string) *events.TaskState {
	return s.taskProj.GetState(taskID)
}

// GetAllTaskStates returns all task states from the projection.
func (s *EventSourcedAuditService) GetAllTaskStates() map[string]*events.TaskState {
	return s.taskProj.GetAllStates()
}

// GetCompletionVelocity returns tasks completed per day.
func (s *EventSourcedAuditService) GetCompletionVelocity() float64 {
	return s.velProj.GetCompletionVelocity()
}

// GetVerificationVelocity returns tasks verified per day.
func (s *EventSourcedAuditService) GetVerificationVelocity() float64 {
	return s.velProj.GetVerificationVelocity()
}

// VerifyIntegrity checks the hash chain for tampering.
func (s *EventSourcedAuditService) VerifyIntegrity() ([]string, error) {
	evts, err := s.store.LoadAll()
	if err != nil {
		return nil, err
	}

	var violations []string
	lastHash := ""

	for i, e := range evts {
		if e.PrevHash != lastHash {
			violations = append(violations, "Event "+e.ID+": PrevHash mismatch at index "+string(rune('0'+i)))
		}

		expected := e.CalculateHash()
		if e.Hash != expected {
			violations = append(violations, "Event "+e.ID+": Hash mismatch - possible tampering")
		}

		lastHash = e.Hash
	}

	return violations, nil
}

// LoadEvents returns all events from the store.
func (s *EventSourcedAuditService) LoadEvents() ([]*events.BaseEvent, error) {
	return s.store.LoadAll()
}

// LoadEventsSince returns events since the given time.
func (s *EventSourcedAuditService) LoadEventsSince(since time.Time) ([]*events.BaseEvent, error) {
	return s.store.LoadSince(since)
}
