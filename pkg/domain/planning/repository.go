package planning

// PlanRepository handles persistence of execution plans.
type PlanRepository interface {
	Save(plan *Plan) error
	Load() (*Plan, error)
}

// StateRepository handles persistence of execution state.
type StateRepository interface {
	Save(state *ExecutionState) error
	Load() (*ExecutionState, error)
}
