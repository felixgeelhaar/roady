package watch_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/watch"
)

func TestPatternFilter_IncludeOnly(t *testing.T) {
	f := watch.NewPatternFilter([]string{"*.md", "*.txt"}, nil)

	tests := []struct {
		path  string
		match bool
	}{
		{"docs/README.md", true},
		{"notes.txt", true},
		{"main.go", false},
		{"src/app.js", false},
	}

	for _, tt := range tests {
		if got := f.Matches(tt.path); got != tt.match {
			t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.match)
		}
	}
}

func TestPatternFilter_ExcludeOnly(t *testing.T) {
	f := watch.NewPatternFilter(nil, []string{"*.tmp", "*.log"})

	tests := []struct {
		path  string
		match bool
	}{
		{"docs/README.md", true},
		{"output.tmp", false},
		{"debug.log", false},
		{"main.go", true},
	}

	for _, tt := range tests {
		if got := f.Matches(tt.path); got != tt.match {
			t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.match)
		}
	}
}

func TestPatternFilter_IncludeAndExclude(t *testing.T) {
	f := watch.NewPatternFilter([]string{"*.md"}, []string{"CHANGELOG.md"})

	tests := []struct {
		path  string
		match bool
	}{
		{"README.md", true},
		{"CHANGELOG.md", false},
		{"main.go", false},
	}

	for _, tt := range tests {
		if got := f.Matches(tt.path); got != tt.match {
			t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.match)
		}
	}
}

func TestPatternFilter_NoPatterns(t *testing.T) {
	f := watch.NewPatternFilter(nil, nil)

	if !f.Matches("anything.txt") {
		t.Error("empty filter should match everything")
	}
}
