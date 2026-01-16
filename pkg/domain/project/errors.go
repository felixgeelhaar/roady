package project

import "errors"

// Domain errors for project coordination.
var (
	// ErrNoPlan indicates no plan exists for the project.
	ErrNoPlan = errors.New("no plan found")

	// ErrPlanNotApproved indicates the plan has not been approved.
	ErrPlanNotApproved = errors.New("plan is not approved")

	// ErrTaskNotFound indicates the task does not exist in the plan.
	ErrTaskNotFound = errors.New("task not found in plan")

	// ErrDependenciesNotMet indicates task dependencies are not completed.
	ErrDependenciesNotMet = errors.New("task dependencies not met")

	// ErrInvalidTransition indicates the requested status transition is not allowed.
	ErrInvalidTransition = errors.New("invalid status transition")

	// ErrTaskAlreadyStarted indicates the task has already been started.
	ErrTaskAlreadyStarted = errors.New("task already started")

	// ErrTaskNotStarted indicates the task has not been started yet.
	ErrTaskNotStarted = errors.New("task not started")

	// ErrNoState indicates no execution state exists.
	ErrNoState = errors.New("no execution state found")

	// ErrOwnerRequired indicates an owner is required for this operation.
	ErrOwnerRequired = errors.New("owner required")

	// ErrEvidenceRequired indicates evidence is required for this operation.
	ErrEvidenceRequired = errors.New("evidence required")
)

// DependencyError provides details about which dependency is blocking.
type DependencyError struct {
	TaskID       string
	DependencyID string
	Status       string
}

func (e *DependencyError) Error() string {
	return "task " + e.TaskID + " blocked by dependency " + e.DependencyID + " (status: " + e.Status + ")"
}

// Is allows errors.Is to work with DependencyError.
func (e *DependencyError) Is(target error) bool {
	return target == ErrDependenciesNotMet
}

// TransitionError provides details about an invalid transition.
type TransitionError struct {
	TaskID     string
	FromStatus string
	ToStatus   string
	Event      string
}

func (e *TransitionError) Error() string {
	return "cannot transition task " + e.TaskID + " from " + e.FromStatus + " to " + e.ToStatus + " via " + e.Event
}

// Is allows errors.Is to work with TransitionError.
func (e *TransitionError) Is(target error) bool {
	return target == ErrInvalidTransition
}
