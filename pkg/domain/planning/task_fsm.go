package planning

import (
	"fmt"

	"github.com/felixgeelhaar/statekit"
)

// TaskState represents the valid states for a task.
const (
	StatePending    = "pending"
	StateInProgress = "in_progress"
	StateBlocked    = "blocked"
	StateDone       = "done"
	StateVerified   = "verified"
)

// TaskContext carries state data.
type TaskContext struct {
	TaskID string
	Guard  func(taskID string, event string) bool
}

// TaskStateMachine defines the valid transitions and rules.
type TaskStateMachine struct {
	interpreter *statekit.Interpreter[TaskContext]
}

func NewTaskStateMachine(initialState string, taskID string, guard func(string, string) bool) (*TaskStateMachine, error) {
	if guard == nil {
		guard = func(string, string) bool { return true }
	}

	// Define the machine
	builder := statekit.NewMachine[TaskContext]("task-machine").
		WithInitial(statekit.StateID(initialState)).
		WithContext(TaskContext{
			TaskID: taskID,
			Guard:  guard,
		}).
		WithGuard("policyGuard", func(ctx TaskContext, e statekit.Event) bool {
			return ctx.Guard(ctx.TaskID, string(e.Type))
		})

	// Pending State
	builder.State(StatePending).
		On("start").Target(StateInProgress).Guard("policyGuard").
		On("block").Target(StateBlocked).
		Done()

	// In Progress State
	builder.State(StateInProgress).
		On("complete").Target(StateDone).
		On("block").Target(StateBlocked).
		On("stop").Target(StatePending).
		Done()

	// Blocked State
	builder.State(StateBlocked).
		On("unblock").Target(StatePending).
		Done()

	// Done State (Final)
	builder.State(StateDone).
		// Removed Type() call as it seems unavailable in this version
		On("reopen").Target(StatePending).
		On("verify").Target(StateVerified).Guard("policyGuard").
		Done()

	// Verified State
	builder.State(StateVerified).
		On("reopen").Target(StatePending).
		Done()

	machine, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build state machine: %w", err)
	}

	interpreter := statekit.NewInterpreter(machine)
	interpreter.Start()

	return &TaskStateMachine{interpreter: interpreter}, nil
}

// Transition attempts to move the task to a new state.
func (sm *TaskStateMachine) Transition(event string) error {
	before := sm.Current()
	// Send event
	sm.interpreter.Send(statekit.Event{Type: statekit.EventType(event)})
	after := sm.Current()

	if before != after {
		return nil
	}

	// If state didn't change, check if the event was valid for the state
	// In statekit, if no transition matches OR guard fails, state stays the same.
	// For simplicity in this investigation, we'll assume if it didn't change, 
	// it was either invalid or blocked.
	return fmt.Errorf("the action '%s' is not allowed while the task is in the '%s' state. Please check the task status or ensure all project policies are met.", event, before)
}

func (sm *TaskStateMachine) Current() string {
	return string(sm.interpreter.State().Value)
}
