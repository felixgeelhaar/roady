package planning

// PrioritySuggestion represents an AI suggestion to change a task's priority.
type PrioritySuggestion struct {
	TaskID          string `json:"task_id"`
	CurrentPriority string `json:"current_priority"`
	SuggestedPriority string `json:"suggested_priority"`
	Reason          string `json:"reason"`
}

// PrioritySuggestions is the result of an AI priority analysis.
type PrioritySuggestions struct {
	Suggestions []PrioritySuggestion `json:"suggestions"`
	Summary     string               `json:"summary"`
}
