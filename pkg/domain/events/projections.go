package events

import (
	"sync"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// =============================================================================
// Task State Projection
// =============================================================================

// TaskStateProjection maintains the current state of all tasks.
type TaskStateProjection struct {
	mu     sync.RWMutex
	states map[string]*TaskState
}

// TaskState represents the current state of a task.
type TaskState struct {
	TaskID       string
	Status       planning.TaskStatus
	Owner        string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	VerifiedAt   *time.Time
	BlockedAt    *time.Time
	ExternalRefs map[string]ExternalRefState
}

// ExternalRefState tracks linked external references.
type ExternalRefState struct {
	Provider   string
	ExternalID string
	URL        string
	LinkedAt   time.Time
}

// NewTaskStateProjection creates a new task state projection.
func NewTaskStateProjection() *TaskStateProjection {
	return &TaskStateProjection{
		states: make(map[string]*TaskState),
	}
}

func (p *TaskStateProjection) Name() string { return "task_state" }

func (p *TaskStateProjection) Apply(event *BaseEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch event.Type {
	case EventTypeTaskStarted:
		taskID := event.Metadata["task_id"].(string)
		state := p.getOrCreate(taskID)
		state.Status = planning.StatusInProgress
		state.Owner = event.Actor
		ts := event.Timestamp
		state.StartedAt = &ts

	case EventTypeTaskCompleted:
		taskID := event.Metadata["task_id"].(string)
		state := p.getOrCreate(taskID)
		state.Status = planning.StatusDone
		ts := event.Timestamp
		state.CompletedAt = &ts

	case EventTypeTaskVerified:
		taskID := event.Metadata["task_id"].(string)
		state := p.getOrCreate(taskID)
		state.Status = planning.StatusVerified
		ts := event.Timestamp
		state.VerifiedAt = &ts

	case EventTypeTaskBlocked:
		taskID := event.Metadata["task_id"].(string)
		state := p.getOrCreate(taskID)
		state.Status = planning.StatusBlocked
		ts := event.Timestamp
		state.BlockedAt = &ts

	case EventTypeTaskUnblocked:
		taskID := event.Metadata["task_id"].(string)
		state := p.getOrCreate(taskID)
		state.Status = planning.StatusPending
		state.BlockedAt = nil

	case EventTypeTaskTransitioned:
		taskID := event.Metadata["task_id"].(string)
		state := p.getOrCreate(taskID)
		if toStatus, ok := event.Metadata["to_status"].(string); ok {
			state.Status = planning.TaskStatus(toStatus)
		}

	case EventTypeExternalRefLinked:
		taskID := event.Metadata["task_id"].(string)
		state := p.getOrCreate(taskID)
		if state.ExternalRefs == nil {
			state.ExternalRefs = make(map[string]ExternalRefState)
		}
		provider := event.Metadata["provider"].(string)
		state.ExternalRefs[provider] = ExternalRefState{
			Provider:   provider,
			ExternalID: event.Metadata["external_id"].(string),
			URL:        getStringMetadata(event.Metadata, "url"),
			LinkedAt:   event.Timestamp,
		}
	}

	return nil
}

func (p *TaskStateProjection) Rebuild(events []*BaseEvent) error {
	p.Reset()
	for _, event := range events {
		if err := p.Apply(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *TaskStateProjection) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.states = make(map[string]*TaskState)
	return nil
}

func (p *TaskStateProjection) getOrCreate(taskID string) *TaskState {
	if state, ok := p.states[taskID]; ok {
		return state
	}
	state := &TaskState{
		TaskID: taskID,
		Status: planning.StatusPending,
	}
	p.states[taskID] = state
	return state
}

// GetState returns the current state for a task.
func (p *TaskStateProjection) GetState(taskID string) *TaskState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.states[taskID]
}

// GetAllStates returns all task states.
func (p *TaskStateProjection) GetAllStates() map[string]*TaskState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string]*TaskState, len(p.states))
	for k, v := range p.states {
		result[k] = v
	}
	return result
}

// =============================================================================
// Velocity Projection
// =============================================================================

// VelocityProjection tracks task completion velocity over time.
type VelocityProjection struct {
	mu           sync.RWMutex
	completions  []time.Time
	verifications []time.Time
	windowDays   int
}

// NewVelocityProjection creates a velocity projection with a rolling window.
func NewVelocityProjection(windowDays int) *VelocityProjection {
	if windowDays <= 0 {
		windowDays = 7
	}
	return &VelocityProjection{
		completions:   make([]time.Time, 0),
		verifications: make([]time.Time, 0),
		windowDays:    windowDays,
	}
}

func (p *VelocityProjection) Name() string { return "velocity" }

func (p *VelocityProjection) Apply(event *BaseEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch event.Type {
	case EventTypeTaskCompleted:
		p.completions = append(p.completions, event.Timestamp)
	case EventTypeTaskVerified:
		p.verifications = append(p.verifications, event.Timestamp)
	}

	return nil
}

func (p *VelocityProjection) Rebuild(events []*BaseEvent) error {
	p.Reset()
	for _, event := range events {
		if err := p.Apply(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *VelocityProjection) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completions = make([]time.Time, 0)
	p.verifications = make([]time.Time, 0)
	return nil
}

// GetCompletionVelocity returns tasks completed per day in the window.
func (p *VelocityProjection) GetCompletionVelocity() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.calculateVelocity(p.completions)
}

// GetVerificationVelocity returns tasks verified per day in the window.
func (p *VelocityProjection) GetVerificationVelocity() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.calculateVelocity(p.verifications)
}

func (p *VelocityProjection) calculateVelocity(timestamps []time.Time) float64 {
	if len(timestamps) == 0 {
		return 0
	}

	cutoff := time.Now().AddDate(0, 0, -p.windowDays)
	count := 0
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			count++
		}
	}

	return float64(count) / float64(p.windowDays)
}

// =============================================================================
// Audit Timeline Projection
// =============================================================================

// AuditTimelineProjection maintains a timeline view of all events.
type AuditTimelineProjection struct {
	mu       sync.RWMutex
	timeline []TimelineEntry
}

// TimelineEntry represents a single entry in the audit timeline.
type TimelineEntry struct {
	Timestamp   time.Time
	EventType   string
	Actor       string
	Description string
	AggregateID string
	Metadata    map[string]interface{}
}

// NewAuditTimelineProjection creates a new timeline projection.
func NewAuditTimelineProjection() *AuditTimelineProjection {
	return &AuditTimelineProjection{
		timeline: make([]TimelineEntry, 0),
	}
}

func (p *AuditTimelineProjection) Name() string { return "audit_timeline" }

func (p *AuditTimelineProjection) Apply(event *BaseEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry := TimelineEntry{
		Timestamp:   event.Timestamp,
		EventType:   event.Type,
		Actor:       event.Actor,
		AggregateID: event.AggregateID_,
		Metadata:    event.Metadata,
		Description: describeEvent(event),
	}

	p.timeline = append(p.timeline, entry)
	return nil
}

func (p *AuditTimelineProjection) Rebuild(events []*BaseEvent) error {
	p.Reset()
	for _, event := range events {
		if err := p.Apply(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *AuditTimelineProjection) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.timeline = make([]TimelineEntry, 0)
	return nil
}

// GetTimeline returns all timeline entries.
func (p *AuditTimelineProjection) GetTimeline() []TimelineEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]TimelineEntry, len(p.timeline))
	copy(result, p.timeline)
	return result
}

// GetRecentEntries returns the most recent n entries.
func (p *AuditTimelineProjection) GetRecentEntries(n int) []TimelineEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if n >= len(p.timeline) {
		result := make([]TimelineEntry, len(p.timeline))
		copy(result, p.timeline)
		return result
	}

	start := len(p.timeline) - n
	result := make([]TimelineEntry, n)
	copy(result, p.timeline[start:])
	return result
}

// =============================================================================
// Helpers
// =============================================================================

func getStringMetadata(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func describeEvent(event *BaseEvent) string {
	switch event.Type {
	case EventTypePlanCreated:
		return "Plan created"
	case EventTypePlanApproved:
		return "Plan approved"
	case EventTypePlanRejected:
		return "Plan rejected"
	case EventTypeTaskStarted:
		return "Task started: " + getStringMetadata(event.Metadata, "task_id")
	case EventTypeTaskCompleted:
		return "Task completed: " + getStringMetadata(event.Metadata, "task_id")
	case EventTypeTaskVerified:
		return "Task verified: " + getStringMetadata(event.Metadata, "task_id")
	case EventTypeTaskBlocked:
		return "Task blocked: " + getStringMetadata(event.Metadata, "task_id")
	case EventTypeTaskUnblocked:
		return "Task unblocked: " + getStringMetadata(event.Metadata, "task_id")
	case EventTypeSyncCompleted:
		return "Sync completed with " + getStringMetadata(event.Metadata, "provider")
	case EventTypeDriftDetected:
		return "Drift detected"
	case EventTypeDriftResolved:
		return "Drift resolved"
	default:
		return event.Type
	}
}
