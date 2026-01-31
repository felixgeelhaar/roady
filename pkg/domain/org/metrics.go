package org

// ProjectMetrics holds computed metrics for a single project.
type ProjectMetrics struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Progress float64 `json:"progress"`
	Verified int     `json:"verified"`
	WIP      int     `json:"wip"`
	Done     int     `json:"done"`
	Pending  int     `json:"pending"`
	Blocked  int     `json:"blocked"`
	Total    int     `json:"total"`
	HasDrift   bool `json:"has_drift"`
	DriftCount int  `json:"drift_count,omitempty"`
}

// ProjectDriftSummary holds drift information for a single project.
type ProjectDriftSummary struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IssueCount int    `json:"issue_count"`
	HasDrift   bool   `json:"has_drift"`
}

// CrossDriftReport aggregates drift across all projects.
type CrossDriftReport struct {
	Projects    []ProjectDriftSummary `json:"projects"`
	TotalIssues int                   `json:"total_issues"`
}

// OrgMetrics aggregates metrics across all projects in an organization.
type OrgMetrics struct {
	OrgName       string           `json:"org_name,omitempty"`
	Projects      []ProjectMetrics `json:"projects"`
	TotalProjects int              `json:"total_projects"`
	TotalTasks    int              `json:"total_tasks"`
	TotalVerified int              `json:"total_verified"`
	TotalWIP      int              `json:"total_wip"`
	AvgProgress   float64          `json:"avg_progress"`
}
