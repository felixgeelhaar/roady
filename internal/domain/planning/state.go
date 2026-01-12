package planning

import "time"

// ExecutionState represents the current reality of the project execution.
type ExecutionState struct {
	ProjectID  string                `json:"project_id"`
	TaskStates map[string]TaskResult `json:"task_states"` // TaskID -> Result
	UpdatedAt  time.Time             `json:"updated_at"`
}

// TaskResult captures the progress of a single task.
type TaskResult struct {
	Status   TaskStatus `json:"status"`
	Path     string     `json:"path"`
	Owner    string     `json:"owner,omitempty"`    // Who is currently working on this?
	Evidence []string   `json:"evidence,omitempty"` // List of evidence (commit hashes, links, etc.)
}

func NewExecutionState(projectID string) *ExecutionState {
	return &ExecutionState{
		ProjectID:  projectID,
		TaskStates: make(map[string]TaskResult),
		UpdatedAt:  time.Now(),
	}
}
