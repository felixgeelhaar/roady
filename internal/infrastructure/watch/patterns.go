package watch

import (
	"path/filepath"
)

// PatternFilter filters file paths based on include/exclude glob patterns.
type PatternFilter struct {
	Include []string
	Exclude []string
}

// NewPatternFilter creates a new pattern filter.
func NewPatternFilter(include, exclude []string) *PatternFilter {
	return &PatternFilter{
		Include: include,
		Exclude: exclude,
	}
}

// Matches returns true if the path passes the filter.
// If include patterns are set, at least one must match.
// If exclude patterns are set, none must match.
func (f *PatternFilter) Matches(path string) bool {
	base := filepath.Base(path)

	// Check excludes first
	for _, pattern := range f.Exclude {
		if matched, _ := filepath.Match(pattern, base); matched {
			return false
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return false
		}
	}

	// If no include patterns, everything passes
	if len(f.Include) == 0 {
		return true
	}

	// At least one include must match
	for _, pattern := range f.Include {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}

	return false
}
