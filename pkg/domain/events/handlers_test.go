package events

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// Mock implementations for testing

type mockDependencyChecker struct {
	unlockedTasks []string
	err           error
}

func (m *mockDependencyChecker) FindUnlockedTasks(ctx context.Context, completedTaskID string) ([]string, error) {
	return m.unlockedTasks, m.err
}

type mockStateInitializer struct {
	initCalled bool
	lastPlanID string
	err        error
}

func (m *mockStateInitializer) InitializeTaskStates(ctx context.Context, planID string) error {
	m.initCalled = true
	m.lastPlanID = planID
	return m.err
}

type mockNotifier struct {
	notifications []notification
}

type notification struct {
	level   NotificationLevel
	title   string
	message string
}

func (m *mockNotifier) Notify(ctx context.Context, level NotificationLevel, title, message string) error {
	m.notifications = append(m.notifications, notification{level, title, message})
	return nil
}

func TestDependencyUnlockHandler_Handle(t *testing.T) {
	checker := &mockDependencyChecker{unlockedTasks: []string{"task-2", "task-3"}}
	notifier := &mockNotifier{}
	handler := NewDependencyUnlockHandler(checker, notifier, slog.Default())

	event := &TaskCompleted{
		BaseEvent: BaseEvent{
			Type:           EventTypeTaskCompleted,
			AggregateID_:   "task-1",
			AggregateType_: AggregateTypeTask,
			Timestamp:      time.Now(),
		},
		TaskID: "task-1",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	if len(notifier.notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifier.notifications))
	}

	if notifier.notifications[0].level != NotificationLevelInfo {
		t.Errorf("Expected info level, got %v", notifier.notifications[0].level)
	}
}

func TestDependencyUnlockHandler_NoUnlockedTasks(t *testing.T) {
	checker := &mockDependencyChecker{unlockedTasks: []string{}}
	notifier := &mockNotifier{}
	handler := NewDependencyUnlockHandler(checker, notifier, slog.Default())

	event := &TaskCompleted{
		BaseEvent: BaseEvent{Type: EventTypeTaskCompleted},
		TaskID:    "task-1",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	if len(notifier.notifications) != 0 {
		t.Errorf("Expected no notifications, got %d", len(notifier.notifications))
	}
}

func TestDependencyUnlockHandler_CheckerError(t *testing.T) {
	checker := &mockDependencyChecker{err: errors.New("checker error")}
	handler := NewDependencyUnlockHandler(checker, nil, slog.Default())

	event := &TaskCompleted{
		BaseEvent: BaseEvent{Type: EventTypeTaskCompleted},
		TaskID:    "task-1",
	}

	err := handler.Handle(context.Background(), event)
	if err == nil {
		t.Error("Expected error from handler")
	}
}

func TestDependencyUnlockHandler_WrongEventType(t *testing.T) {
	checker := &mockDependencyChecker{}
	handler := NewDependencyUnlockHandler(checker, nil, slog.Default())

	// Send a different event type
	event := &PlanApproved{
		BaseEvent: BaseEvent{Type: EventTypePlanApproved},
		PlanID:    "plan-1",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Expected nil for wrong event type, got: %v", err)
	}
}

func TestDependencyUnlockHandler_Registration(t *testing.T) {
	handler := NewDependencyUnlockHandler(nil, nil, nil)
	reg := handler.Registration()

	if reg.Name != "DependencyUnlockHandler" {
		t.Errorf("Expected name DependencyUnlockHandler, got %s", reg.Name)
	}

	if len(reg.EventTypes) != 1 || reg.EventTypes[0] != EventTypeTaskCompleted {
		t.Error("Expected registration for task.completed event")
	}
}

func TestDriftWarningHandler_Handle(t *testing.T) {
	notifier := &mockNotifier{}
	handler := NewDriftWarningHandler(notifier, slog.Default())

	event := &DriftDetected{
		BaseEvent: BaseEvent{
			Type:           EventTypeDriftDetected,
			AggregateID_:   "project-1",
			AggregateType_: AggregateTypePlan,
			Timestamp:      time.Now(),
		},
		IssueCount: 3,
		Severities: []string{"warning", "error"},
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	if len(notifier.notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifier.notifications))
	}
}

func TestDriftWarningHandler_CriticalSeverity(t *testing.T) {
	notifier := &mockNotifier{}
	handler := NewDriftWarningHandler(notifier, slog.Default())

	event := &DriftDetected{
		BaseEvent:  BaseEvent{Type: EventTypeDriftDetected},
		IssueCount: 1,
		Severities: []string{"critical"},
	}

	_ = handler.Handle(context.Background(), event)

	if len(notifier.notifications) != 1 {
		t.Fatal("Expected 1 notification")
	}

	if notifier.notifications[0].level != NotificationLevelError {
		t.Errorf("Expected error level for critical severity, got %v", notifier.notifications[0].level)
	}
}

func TestDriftWarningHandler_Registration(t *testing.T) {
	handler := NewDriftWarningHandler(nil, nil)
	reg := handler.Registration()

	if reg.Name != "DriftWarningHandler" {
		t.Errorf("Expected name DriftWarningHandler, got %s", reg.Name)
	}

	if len(reg.EventTypes) != 1 || reg.EventTypes[0] != EventTypeDriftDetected {
		t.Error("Expected registration for drift.detected event")
	}
}

func TestStateProjectionHandler_Handle(t *testing.T) {
	initializer := &mockStateInitializer{}
	handler := NewStateProjectionHandler(initializer, slog.Default())

	event := &PlanApproved{
		BaseEvent: BaseEvent{
			Type:           EventTypePlanApproved,
			AggregateID_:   "plan-1",
			AggregateType_: AggregateTypePlan,
			Timestamp:      time.Now(),
		},
		PlanID:   "plan-1",
		Approver: "user-1",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	if !initializer.initCalled {
		t.Error("Expected InitializeTaskStates to be called")
	}

	if initializer.lastPlanID != "plan-1" {
		t.Errorf("Expected plan ID 'plan-1', got '%s'", initializer.lastPlanID)
	}
}

func TestStateProjectionHandler_InitializerError(t *testing.T) {
	initializer := &mockStateInitializer{err: errors.New("init error")}
	handler := NewStateProjectionHandler(initializer, slog.Default())

	event := &PlanApproved{
		BaseEvent: BaseEvent{Type: EventTypePlanApproved},
		PlanID:    "plan-1",
	}

	err := handler.Handle(context.Background(), event)
	if err == nil {
		t.Error("Expected error from handler")
	}
}

func TestStateProjectionHandler_Registration(t *testing.T) {
	handler := NewStateProjectionHandler(nil, nil)
	reg := handler.Registration()

	if reg.Name != "StateProjectionHandler" {
		t.Errorf("Expected name StateProjectionHandler, got %s", reg.Name)
	}

	if len(reg.EventTypes) != 1 || reg.EventTypes[0] != EventTypePlanApproved {
		t.Error("Expected registration for plan.approved event")
	}
}

func TestLoggingHandler_Handle(t *testing.T) {
	handler := NewLoggingHandler(slog.Default())

	event := &PlanCreated{
		BaseEvent: BaseEvent{
			Type:           EventTypePlanCreated,
			AggregateID_:   "plan-1",
			AggregateType_: AggregateTypePlan,
			Timestamp:      time.Now(),
			Version_:       1,
		},
		PlanID: "plan-1",
	}

	// Should not return error
	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
}

func TestLoggingHandler_Registration(t *testing.T) {
	handler := NewLoggingHandler(nil)
	reg := handler.Registration()

	if reg.Name != "LoggingHandler" {
		t.Errorf("Expected name LoggingHandler, got %s", reg.Name)
	}

	if len(reg.EventTypes) != 1 || reg.EventTypes[0] != "*" {
		t.Error("Expected wildcard registration")
	}
}

func TestTaskTransitionHandler_BlockedCallback(t *testing.T) {
	handler := NewTaskTransitionHandler(slog.Default())

	blockedCalled := false
	handler.OnBlocked = func(ctx context.Context, taskID string, fromStatus planning.TaskStatus) error {
		blockedCalled = true
		if taskID != "task-1" {
			t.Errorf("Expected task ID 'task-1', got '%s'", taskID)
		}
		if fromStatus != planning.StatusInProgress {
			t.Errorf("Expected from status 'in_progress', got '%s'", fromStatus)
		}
		return nil
	}

	event := &TaskTransitioned{
		BaseEvent:  BaseEvent{Type: EventTypeTaskTransitioned},
		TaskID:     "task-1",
		FromStatus: planning.StatusInProgress,
		ToStatus:   planning.StatusBlocked,
	}

	_ = handler.Handle(context.Background(), event)

	if !blockedCalled {
		t.Error("Expected OnBlocked callback to be called")
	}
}

func TestTaskTransitionHandler_UnblockedCallback(t *testing.T) {
	handler := NewTaskTransitionHandler(slog.Default())

	unblockedCalled := false
	handler.OnUnblocked = func(ctx context.Context, taskID string, toStatus planning.TaskStatus) error {
		unblockedCalled = true
		return nil
	}

	event := &TaskTransitioned{
		BaseEvent:  BaseEvent{Type: EventTypeTaskTransitioned},
		TaskID:     "task-1",
		FromStatus: planning.StatusBlocked,
		ToStatus:   planning.StatusPending,
	}

	_ = handler.Handle(context.Background(), event)

	if !unblockedCalled {
		t.Error("Expected OnUnblocked callback to be called")
	}
}

func TestTaskTransitionHandler_Registration(t *testing.T) {
	handler := NewTaskTransitionHandler(nil)
	reg := handler.Registration()

	if reg.Name != "TaskTransitionHandler" {
		t.Errorf("Expected name TaskTransitionHandler, got %s", reg.Name)
	}

	if len(reg.EventTypes) != 1 || reg.EventTypes[0] != EventTypeTaskTransitioned {
		t.Error("Expected registration for task.transitioned event")
	}
}

func TestDependencyUnlockHandler_NilChecker(t *testing.T) {
	handler := NewDependencyUnlockHandler(nil, nil, slog.Default())

	event := &TaskCompleted{
		BaseEvent: BaseEvent{Type: EventTypeTaskCompleted},
		TaskID:    "task-1",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle should return nil when checker is nil: %v", err)
	}
}

func TestDependencyUnlockHandler_SingleUnlockedTask(t *testing.T) {
	checker := &mockDependencyChecker{unlockedTasks: []string{"task-2"}}
	notifier := &mockNotifier{}
	handler := NewDependencyUnlockHandler(checker, notifier, slog.Default())

	event := &TaskCompleted{
		BaseEvent: BaseEvent{Type: EventTypeTaskCompleted},
		TaskID:    "task-1",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	if len(notifier.notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifier.notifications))
	}

	// Single task should use singular message
	msg := notifier.notifications[0].message
	if msg != "Task task-2 is now ready to start after task-1 was completed." {
		t.Errorf("unexpected message: %s", msg)
	}
}

func TestDependencyUnlockHandler_NilNotifier(t *testing.T) {
	checker := &mockDependencyChecker{unlockedTasks: []string{"task-2"}}
	handler := NewDependencyUnlockHandler(checker, nil, slog.Default())

	event := &TaskCompleted{
		BaseEvent: BaseEvent{Type: EventTypeTaskCompleted},
		TaskID:    "task-1",
	}

	// Should not panic even with nil notifier
	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
}

func TestDriftWarningHandler_WrongEventType(t *testing.T) {
	handler := NewDriftWarningHandler(nil, slog.Default())

	event := &PlanCreated{
		BaseEvent: BaseEvent{Type: EventTypePlanCreated},
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle should return nil for wrong event type: %v", err)
	}
}

func TestDriftWarningHandler_ZeroIssues(t *testing.T) {
	notifier := &mockNotifier{}
	handler := NewDriftWarningHandler(notifier, slog.Default())

	event := &DriftDetected{
		BaseEvent:  BaseEvent{Type: EventTypeDriftDetected},
		IssueCount: 0,
		Severities: nil,
	}

	_ = handler.Handle(context.Background(), event)

	// Zero issues should not trigger notification
	if len(notifier.notifications) != 0 {
		t.Errorf("Expected 0 notifications for 0 issues, got %d", len(notifier.notifications))
	}
}

func TestDriftWarningHandler_NilNotifier(t *testing.T) {
	handler := NewDriftWarningHandler(nil, slog.Default())

	event := &DriftDetected{
		BaseEvent:  BaseEvent{Type: EventTypeDriftDetected},
		IssueCount: 5,
		Severities: []string{"warning"},
	}

	// Should not panic with nil notifier
	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
}

func TestStateProjectionHandler_WrongEventType(t *testing.T) {
	handler := NewStateProjectionHandler(nil, slog.Default())

	event := &TaskCompleted{
		BaseEvent: BaseEvent{Type: EventTypeTaskCompleted},
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle should return nil for wrong event type: %v", err)
	}
}

func TestStateProjectionHandler_NilInitializer(t *testing.T) {
	handler := NewStateProjectionHandler(nil, slog.Default())

	event := &PlanApproved{
		BaseEvent: BaseEvent{Type: EventTypePlanApproved},
		PlanID:    "plan-1",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle should return nil when initializer is nil: %v", err)
	}
}

func TestTaskTransitionHandler_WrongEventType(t *testing.T) {
	handler := NewTaskTransitionHandler(slog.Default())

	event := &PlanCreated{
		BaseEvent: BaseEvent{Type: EventTypePlanCreated},
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle should return nil for wrong event type: %v", err)
	}
}

func TestTaskTransitionHandler_NoCallbacks(t *testing.T) {
	handler := NewTaskTransitionHandler(slog.Default())

	// No OnBlocked or OnUnblocked callbacks set
	event := &TaskTransitioned{
		BaseEvent:  BaseEvent{Type: EventTypeTaskTransitioned},
		TaskID:     "task-1",
		FromStatus: planning.StatusInProgress,
		ToStatus:   planning.StatusBlocked,
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed with no callbacks: %v", err)
	}
}

func TestTaskTransitionHandler_NormalTransition(t *testing.T) {
	handler := NewTaskTransitionHandler(slog.Default())

	// Transition that is not blocked/unblocked
	event := &TaskTransitioned{
		BaseEvent:  BaseEvent{Type: EventTypeTaskTransitioned},
		TaskID:     "task-1",
		FromStatus: planning.StatusPending,
		ToStatus:   planning.StatusInProgress,
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
}

func TestFormatUnlockedMessage(t *testing.T) {
	single := formatUnlockedMessage("task-1", []string{"task-2"})
	if single != "Task task-2 is now ready to start after task-1 was completed." {
		t.Errorf("unexpected single message: %s", single)
	}

	multiple := formatUnlockedMessage("task-1", []string{"task-2", "task-3"})
	if multiple != "Multiple tasks are now ready to start after task-1 was completed." {
		t.Errorf("unexpected multiple message: %s", multiple)
	}
}

func TestFormatDriftMessage(t *testing.T) {
	single := formatDriftMessage(1, nil)
	if single != "1 drift issue detected in the project." {
		t.Errorf("unexpected single message: %s", single)
	}

	multiple := formatDriftMessage(3, []string{"warning"})
	// Just verify it returns a non-empty string
	if multiple == "" {
		t.Error("expected non-empty drift message")
	}
}

func TestContainsSeverity(t *testing.T) {
	severities := []string{"warning", "error", "critical"}

	if !containsSeverity(severities, "critical") {
		t.Error("Expected to find 'critical'")
	}

	if containsSeverity(severities, "info") {
		t.Error("Did not expect to find 'info'")
	}

	if containsSeverity(nil, "warning") {
		t.Error("Did not expect to find anything in nil slice")
	}
}
