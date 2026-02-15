// Package org provides organizational multi-project domain types.
package org

// SharedPolicy defines org-level policy defaults that projects inherit.
type SharedPolicy struct {
	MaxWIP      int  `yaml:"max_wip,omitempty" json:"max_wip,omitempty"`
	AllowAI     bool `yaml:"allow_ai,omitempty" json:"allow_ai,omitempty"`
	TokenLimit  int  `yaml:"token_limit,omitempty" json:"token_limit,omitempty"`
	BudgetHours int  `yaml:"budget_hours,omitempty" json:"budget_hours,omitempty"`
}

// OrgConfig defines the organizational configuration for multi-repo management.
type OrgConfig struct {
	Name         string        `yaml:"name" json:"name"`
	Repos        []string      `yaml:"repos" json:"repos"`
	SharedPolicy *SharedPolicy `yaml:"shared_policy,omitempty" json:"shared_policy,omitempty"`
}
