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
