package planning_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestTaskStateMachine(t *testing.T) {
	// 1. Init
	fsm, err := planning.NewTaskStateMachine(planning.StatePending, "t1", nil)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if fsm.Current() != planning.StatePending {
		t.Errorf("Expected Pending, got %s", fsm.Current())
	}

	// 2. Transition
	if err := fsm.Transition("start"); err != nil {
		t.Errorf("Start failed: %v", err)
	}
	if fsm.Current() != planning.StateInProgress {
		t.Errorf("Expected InProgress, got %s", fsm.Current())
	}

	// 3. Invalid Transition
	err = fsm.Transition("invalid")
	if err == nil {
		t.Errorf("Expected error on invalid transition")
	}

	// 4. Guarded Transition
	blockedGuard := func(tid string, ev string) bool { return false }
	fsm2, _ := planning.NewTaskStateMachine(planning.StatePending, "t2", blockedGuard)
	err = fsm2.Transition("start")
	if err == nil {
		t.Errorf("Expected error on guarded transition")
	}
	if fsm2.Current() != planning.StatePending {
		t.Errorf("State changed despite failing guard")
	}
}

func TestTaskStateMachine_CurrentStatus(t *testing.T) {
	fsm, err := planning.NewTaskStateMachine(planning.StatePending, "t1", nil)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	status := fsm.CurrentStatus()
	if status != planning.StatusPending {
		t.Errorf("Expected StatusPending, got %v", status)
	}

	if err := fsm.Transition("start"); err != nil {
		t.Fatal(err)
	}
	status = fsm.CurrentStatus()
	if status != planning.StatusInProgress {
		t.Errorf("Expected StatusInProgress, got %v", status)
	}
}

func TestTaskStateMachine_CanTransition(t *testing.T) {
	fsm, _ := planning.NewTaskStateMachine(planning.StatePending, "t1", nil)

	// Valid event for pending
	if !fsm.CanTransition("start") {
		t.Error("Expected CanTransition('start') to be true for pending state")
	}

	// Invalid event for pending
	if fsm.CanTransition("complete") {
		t.Error("Expected CanTransition('complete') to be false for pending state")
	}
}

func TestTaskStateMachine_ValidEvents(t *testing.T) {
	fsm, _ := planning.NewTaskStateMachine(planning.StatePending, "t1", nil)

	events := fsm.ValidEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 valid events for pending, got %d", len(events))
	}

	if err := fsm.Transition("start"); err != nil {
		t.Fatal(err)
	}
	events = fsm.ValidEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 valid events for in_progress, got %d", len(events))
	}
}

func TestTaskStateMachine_IsFinal(t *testing.T) {
	fsm, _ := planning.NewTaskStateMachine(planning.StatePending, "t1", nil)

	if fsm.IsFinal() {
		t.Error("Pending state should not be final")
	}

	if err := fsm.Transition("start"); err != nil {
		t.Fatal(err)
	}
	if err := fsm.Transition("complete"); err != nil {
		t.Fatal(err)
	}
	if fsm.IsFinal() {
		t.Error("Done state should not be final")
	}

	if err := fsm.Transition("verify"); err != nil {
		t.Fatal(err)
	}
	if !fsm.IsFinal() {
		t.Error("Verified state should be final")
	}
}

func TestTaskStateMachine_IsComplete(t *testing.T) {
	fsm, _ := planning.NewTaskStateMachine(planning.StatePending, "t1", nil)

	if fsm.IsComplete() {
		t.Error("Pending state should not be complete")
	}

	if err := fsm.Transition("start"); err != nil {
		t.Fatal(err)
	}
	if fsm.IsComplete() {
		t.Error("InProgress state should not be complete")
	}

	if err := fsm.Transition("complete"); err != nil {
		t.Fatal(err)
	}
	if !fsm.IsComplete() {
		t.Error("Done state should be complete")
	}

	if err := fsm.Transition("verify"); err != nil {
		t.Fatal(err)
	}
	if !fsm.IsComplete() {
		t.Error("Verified state should be complete")
	}
}
