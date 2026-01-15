package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusBlocked    TaskStatus = "blocked"
	StatusDone       TaskStatus = "done"
	StatusVerified   TaskStatus = "verified"
)

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

// Plan represents a derived execution graph from a Spec.
type Plan struct {
	ID             string         `json:"id" yaml:"id"`
	SpecID         string         `json:"spec_id" yaml:"spec_id"` // Links back to the Spec it implements
	Tasks          []Task         `json:"tasks" yaml:"tasks"`
	ApprovalStatus ApprovalStatus `json:"approval_status" yaml:"approval_status"`
	CreatedAt      time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" yaml:"updated_at"`
}

type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

// Task is a unit of work (structural intent).
type Task struct {
	ID          string       `json:"id" yaml:"id"`
	Title       string       `json:"title" yaml:"title"`
	Description string       `json:"description" yaml:"description"`
	Priority    TaskPriority `json:"priority" yaml:"priority"`
	Estimate    string       `json:"estimate" yaml:"estimate"`     // e.g., "4h", "1d"
	DependsOn   []string     `json:"depends_on" yaml:"depends_on"` // IDs of tasks this task depends on
	FeatureID   string       `json:"feature_id" yaml:"feature_id"` // Link to the feature in the spec
}

// Hash returns a deterministic hash of the plan structure.
func (p *Plan) Hash() string {
	h := sha256.New()
	h.Write([]byte(p.ID))
	h.Write([]byte(p.SpecID))
	for _, t := range p.Tasks {
		h.Write([]byte(t.ID))
	}
	return hex.EncodeToString(h.Sum(nil))
}
