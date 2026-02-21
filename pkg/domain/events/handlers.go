package events

import (
	"context"
	"log/slog"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// DependencyChecker is the interface for checking task dependencies.
type DependencyChecker interface {
	// FindUnlockedTasks returns task IDs that became unblocked after a task completion.
	FindUnlockedTasks(ctx context.Context, completedTaskID string) ([]string, error)
}

// StateInitializer is the interface for initializing task states from a plan.
type StateInitializer interface {
	// InitializeTaskStates creates initial TaskResult entries for all tasks in a plan.
	InitializeTaskStates(ctx context.Context, planID string) error
}

// Notifier is the interface for sending notifications.
type Notifier interface {
	// Notify sends a notification with the given level, title, and message.
	Notify(ctx context.Context, level NotificationLevel, title, message string) error
}

// NotificationLevel represents the severity of a notification.
type NotificationLevel string

const (
	NotificationLevelInfo    NotificationLevel = "info"
	NotificationLevelWarning NotificationLevel = "warning"
	NotificationLevelError   NotificationLevel = "error"
)

// DependencyUnlockHandler handles TaskCompleted events to check for newly unlocked tasks.
type DependencyUnlockHandler struct {
	checker  DependencyChecker
	notifier Notifier
	logger   *slog.Logger
}

// NewDependencyUnlockHandler creates a new DependencyUnlockHandler.
func NewDependencyUnlockHandler(checker DependencyChecker, notifier Notifier, logger *slog.Logger) *DependencyUnlockHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DependencyUnlockHandler{
		checker:  checker,
		notifier: notifier,
		logger:   logger,
	}
}

// Handle processes TaskCompleted events and checks for unlocked dependencies.
func (h *DependencyUnlockHandler) Handle(ctx context.Context, event DomainEvent) error {
	completed, ok := event.(*TaskCompleted)
	if !ok {
		return nil // Not a TaskCompleted event
	}

	h.logger.Debug("checking for unlocked dependencies",
		"task_id", completed.TaskID,
		"event_type", event.EventType())

	if h.checker == nil {
		return nil
	}

	unlocked, err := h.checker.FindUnlockedTasks(ctx, completed.TaskID)
	if err != nil {
		h.logger.Error("failed to find unlocked tasks",
			"task_id", completed.TaskID,
			"error", err)
		return err
	}

	if len(unlocked) > 0 {
		h.logger.Info("tasks unlocked after completion",
			"completed_task", completed.TaskID,
			"unlocked_tasks", unlocked)

		if h.notifier != nil {
			_ = h.notifier.Notify(ctx, NotificationLevelInfo,
				"Tasks Unlocked",
				formatUnlockedMessage(completed.TaskID, unlocked))
		}
	}

	return nil
}

// Registration returns the HandlerRegistration for this handler.
func (h *DependencyUnlockHandler) Registration() HandlerRegistration {
	return HandlerRegistration{
		Name:       "DependencyUnlockHandler",
		Handler:    h.Handle,
		EventTypes: []string{EventTypeTaskCompleted},
	}
}

// DriftWarningHandler handles DriftDetected events to log warnings.
type DriftWarningHandler struct {
	notifier Notifier
	logger   *slog.Logger
}

// NewDriftWarningHandler creates a new DriftWarningHandler.
func NewDriftWarningHandler(notifier Notifier, logger *slog.Logger) *DriftWarningHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DriftWarningHandler{
		notifier: notifier,
		logger:   logger,
	}
}

// Handle processes DriftDetected events.
func (h *DriftWarningHandler) Handle(ctx context.Context, event DomainEvent) error {
	drift, ok := event.(*DriftDetected)
	if !ok {
		return nil
	}

	h.logger.Warn("drift detected in project",
		"issue_count", drift.IssueCount,
		"severities", drift.Severities,
		"aggregate_id", drift.AggregateID())

	if h.notifier != nil && drift.IssueCount > 0 {
		level := NotificationLevelWarning
		if containsSeverity(drift.Severities, "critical") {
			level = NotificationLevelError
		}

		_ = h.notifier.Notify(ctx, level,
			"Drift Detected",
			formatDriftMessage(drift.IssueCount, drift.Severities))
	}

	return nil
}

// Registration returns the HandlerRegistration for this handler.
func (h *DriftWarningHandler) Registration() HandlerRegistration {
	return HandlerRegistration{
		Name:       "DriftWarningHandler",
		Handler:    h.Handle,
		EventTypes: []string{EventTypeDriftDetected},
	}
}

// StateProjectionHandler handles PlanApproved events to initialize task states.
type StateProjectionHandler struct {
	initializer StateInitializer
	logger      *slog.Logger
}

// NewStateProjectionHandler creates a new StateProjectionHandler.
func NewStateProjectionHandler(initializer StateInitializer, logger *slog.Logger) *StateProjectionHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &StateProjectionHandler{
		initializer: initializer,
		logger:      logger,
	}
}

// Handle processes PlanApproved events.
func (h *StateProjectionHandler) Handle(ctx context.Context, event DomainEvent) error {
	approved, ok := event.(*PlanApproved)
	if !ok {
		return nil
	}

	h.logger.Info("initializing task states for approved plan",
		"plan_id", approved.PlanID,
		"approver", approved.Approver)

	if h.initializer == nil {
		return nil
	}

	if err := h.initializer.InitializeTaskStates(ctx, approved.PlanID); err != nil {
		h.logger.Error("failed to initialize task states",
			"plan_id", approved.PlanID,
			"error", err)
		return err
	}

	return nil
}

// Registration returns the HandlerRegistration for this handler.
func (h *StateProjectionHandler) Registration() HandlerRegistration {
	return HandlerRegistration{
		Name:       "StateProjectionHandler",
		Handler:    h.Handle,
		EventTypes: []string{EventTypePlanApproved},
	}
}

// LoggingHandler is a catch-all handler that logs all events.
type LoggingHandler struct {
	logger *slog.Logger
}

// NewLoggingHandler creates a new LoggingHandler.
func NewLoggingHandler(logger *slog.Logger) *LoggingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingHandler{logger: logger}
}

// Handle logs the event details.
func (h *LoggingHandler) Handle(ctx context.Context, event DomainEvent) error {
	h.logger.Debug("domain event",
		"event_type", event.EventType(),
		"aggregate_id", event.AggregateID(),
		"aggregate_type", event.AggregateType(),
		"version", event.Version(),
		"occurred_at", event.OccurredAt())
	return nil
}

// Registration returns the HandlerRegistration for this handler.
func (h *LoggingHandler) Registration() HandlerRegistration {
	return HandlerRegistration{
		Name:       "LoggingHandler",
		Handler:    h.Handle,
		EventTypes: []string{"*"}, // Wildcard - all events
	}
}

// TaskTransitionHandler handles TaskTransitioned events for status change tracking.
type TaskTransitionHandler struct {
	logger *slog.Logger
	// OnBlocked is called when a task transitions to blocked state
	OnBlocked func(ctx context.Context, taskID string, fromStatus planning.TaskStatus) error
	// OnUnblocked is called when a task transitions from blocked state
	OnUnblocked func(ctx context.Context, taskID string, toStatus planning.TaskStatus) error
}

// NewTaskTransitionHandler creates a new TaskTransitionHandler.
func NewTaskTransitionHandler(logger *slog.Logger) *TaskTransitionHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskTransitionHandler{logger: logger}
}

// Handle processes TaskTransitioned events.
func (h *TaskTransitionHandler) Handle(ctx context.Context, event DomainEvent) error {
	transition, ok := event.(*TaskTransitioned)
	if !ok {
		return nil
	}

	h.logger.Info("task status changed",
		"task_id", transition.TaskID,
		"from_status", transition.FromStatus,
		"to_status", transition.ToStatus)

	// Handle blocked/unblocked callbacks
	if transition.ToStatus == planning.StatusBlocked && h.OnBlocked != nil {
		return h.OnBlocked(ctx, transition.TaskID, transition.FromStatus)
	}

	if transition.FromStatus == planning.StatusBlocked && h.OnUnblocked != nil {
		return h.OnUnblocked(ctx, transition.TaskID, transition.ToStatus)
	}

	return nil
}

// Registration returns the HandlerRegistration for this handler.
func (h *TaskTransitionHandler) Registration() HandlerRegistration {
	return HandlerRegistration{
		Name:       "TaskTransitionHandler",
		Handler:    h.Handle,
		EventTypes: []string{EventTypeTaskTransitioned},
	}
}

// Helper functions

func formatUnlockedMessage(completedTask string, unlockedTasks []string) string {
	if len(unlockedTasks) == 1 {
		return "Task " + unlockedTasks[0] + " is now ready to start after " + completedTask + " was completed."
	}
	return "Multiple tasks are now ready to start after " + completedTask + " was completed."
}

func formatDriftMessage(issueCount int, severities []string) string {
	if issueCount == 1 {
		return "1 drift issue detected in the project."
	}
	return "Multiple drift issues (" + string(rune(issueCount+'0')) + ") detected in the project."
}

func containsSeverity(severities []string, target string) bool {
	for _, s := range severities {
		if s == target {
			return true
		}
	}
	return false
}
