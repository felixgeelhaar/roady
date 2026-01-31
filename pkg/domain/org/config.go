// Package org provides organizational multi-project domain types.
package org

// OrgConfig defines the organizational configuration for multi-repo management.
type OrgConfig struct {
	Name  string   `yaml:"name" json:"name"`
	Repos []string `yaml:"repos" json:"repos"`
}
