package planning

// SmartTask extends Task with codebase-aware fields.
type SmartTask struct {
	Task
	Files      []string `json:"files,omitempty"`
	Complexity string   `json:"complexity,omitempty"` // low, medium, high
}

// SmartPlan is a plan generated with codebase context.
type SmartPlan struct {
	Tasks   []SmartTask `json:"tasks"`
	Summary string      `json:"summary"`
}
