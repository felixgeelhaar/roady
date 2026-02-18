package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ProductSpec represents the top-level specification of what is being built.
type ProductSpec struct {
	ID          string       `json:"id" yaml:"id"`
	Title       string       `json:"title" yaml:"title"`
	Description string       `json:"description" yaml:"description"`
	Features    []Feature    `json:"features" yaml:"features"`
	Constraints []Constraint `json:"constraints" yaml:"constraints"`
	Version     string       `json:"version" yaml:"version"`
}

// Feature represents a specific functional unit within the spec.
type Feature struct {
	ID           string        `json:"id" yaml:"id"`
	Title        string        `json:"title" yaml:"title"`
	Description  string        `json:"description" yaml:"description"`
	Requirements []Requirement `json:"requirements" yaml:"requirements"`
}

// Requirement represents a granular condition that a feature must satisfy.
type Requirement struct {
	ID          string   `json:"id" yaml:"id"`
	Title       string   `json:"title" yaml:"title"`
	Description string   `json:"description" yaml:"description"`
	Priority    string   `json:"priority" yaml:"priority"`
	Estimate    string   `json:"estimate" yaml:"estimate"`
	DependsOn   []string `json:"depends_on" yaml:"depends_on"`
}

// Constraint represents non-functional requirements or policies.
type Constraint struct {
	ID          string `json:"id" yaml:"id"`
	Description string `json:"description" yaml:"description"`
}

// Hash returns a deterministic hash of the spec for drift detection.
func (s *ProductSpec) Hash() string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s:%s:%s", s.ID, s.Version, s.Title)

	// Hash features and descriptions
	for _, f := range s.Features {
		h.Write([]byte(f.ID))
		h.Write([]byte(f.Description))
		for _, r := range f.Requirements {
			h.Write([]byte(r.ID))
			h.Write([]byte(r.Description))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Validate checks the spec for structural integrity.
func (s *ProductSpec) Validate() []error {
	var errs []error
	if s.ID == "" {
		errs = append(errs, fmt.Errorf("spec ID is required"))
	}
	if s.Title == "" {
		errs = append(errs, fmt.Errorf("spec Title is required"))
	}
	if len(s.Features) == 0 {
		errs = append(errs, fmt.Errorf("spec must have at least one feature"))
	}

	seenIDs := make(map[string]bool)
	for i, f := range s.Features {
		if f.ID == "" {
			errs = append(errs, fmt.Errorf("feature at index %d missing ID", i))
		}
		if seenIDs[f.ID] {
			errs = append(errs, fmt.Errorf("duplicate feature ID: %s", f.ID))
		}
		seenIDs[f.ID] = true

		for j, r := range f.Requirements {
			if r.ID == "" {
				errs = append(errs, fmt.Errorf("feature '%s' requirement at index %d missing ID", f.ID, j))
			}
			if r.Title == "" {
				errs = append(errs, fmt.Errorf("feature '%s' requirement '%s' missing title", f.ID, r.ID))
			}
		}
	}
	return errs
}
