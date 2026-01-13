package domain

import (
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

// WorkspaceRepository handles the persistence of roady artifacts in the .roady/ directory.
type WorkspaceRepository interface {
	Initialize() error
	IsInitialized() bool
	SaveSpec(spec *spec.ProductSpec) error
	LoadSpec() (*spec.ProductSpec, error)
	SaveSpecLock(spec *spec.ProductSpec) error
	LoadSpecLock() (*spec.ProductSpec, error)
	SavePlan(plan *planning.Plan) error
	LoadPlan() (*planning.Plan, error)
	SaveState(state *planning.ExecutionState) error
	LoadState() (*planning.ExecutionState, error)
	SavePolicy(cfg *PolicyConfig) error
	LoadPolicy() (*PolicyConfig, error)
	RecordEvent(event Event) error
	LoadEvents() ([]Event, error)
	UpdateUsage(stats UsageStats) error
	LoadUsage() (*UsageStats, error)
}

// PolicyConfig is the serialized representation of policy.yaml
type PolicyConfig struct {
	MaxWIP     int  `yaml:"max_wip"`
	AllowAI    bool `yaml:"allow_ai"`
	TokenLimit int  `yaml:"token_limit"` // Horizon 4: Max tokens allowed for this project
}
