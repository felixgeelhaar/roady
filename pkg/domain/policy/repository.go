package policy

// PolicyConfig is the serialized representation of policy.yaml
type PolicyConfig struct {
	MaxWIP      int  `yaml:"max_wip"`
	AllowAI     bool `yaml:"allow_ai"`
	TokenLimit  int  `yaml:"token_limit"`
	BudgetHours int  `yaml:"budget_hours"`
}

// Repository handles persistence of policy configurations.
type Repository interface {
	Save(cfg *PolicyConfig) error
	Load() (*PolicyConfig, error)
}
