package spec_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestProductSpec_Hash(t *testing.T) {
	s1 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
	}

	s2 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
	}

	s3 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.1",
		Title:   "Test Spec",
	}

	if s1.Hash() != s2.Hash() {
		t.Errorf("Expected hashes to be identical for identical specs")
	}

	if s1.Hash() == s3.Hash() {
		t.Errorf("Expected hashes to differ for different specs")
	}

	s4 := &spec.ProductSpec{ID: "test-1", Title: "Different Title"}
	if s1.Hash() == s4.Hash() {
		t.Error("Expected hashes to differ for different titles")
	}

	s5 := &spec.ProductSpec{ID: "test-1", Version: "2.0"}
	if s1.Hash() == s5.Hash() {
		t.Error("Expected hashes to differ for different versions")
	}
}

func TestProductSpec_HashWithFeatures(t *testing.T) {
	s1 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
		Features: []spec.Feature{
			{
				ID:          "f1",
				Description: "Feature 1",
				Requirements: []spec.Requirement{
					{ID: "r1", Description: "Requirement 1"},
					{ID: "r2", Description: "Requirement 2"},
				},
			},
		},
	}

	s2 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
		Features: []spec.Feature{
			{
				ID:          "f1",
				Description: "Feature 1",
				Requirements: []spec.Requirement{
					{ID: "r1", Description: "Requirement 1"},
					{ID: "r2", Description: "Requirement 2"},
				},
			},
		},
	}

	s3 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
		Features: []spec.Feature{
			{
				ID:          "f1",
				Description: "Feature 1 CHANGED",
				Requirements: []spec.Requirement{
					{ID: "r1", Description: "Requirement 1"},
				},
			},
		},
	}

	if s1.Hash() != s2.Hash() {
		t.Error("identical specs with features should have same hash")
	}

	if s1.Hash() == s3.Hash() {
		t.Error("different feature descriptions should produce different hashes")
	}

	// Test with different requirement descriptions
	s4 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
		Features: []spec.Feature{
			{
				ID:          "f1",
				Description: "Feature 1",
				Requirements: []spec.Requirement{
					{ID: "r1", Description: "CHANGED Requirement 1"},
					{ID: "r2", Description: "Requirement 2"},
				},
			},
		},
	}

	if s1.Hash() == s4.Hash() {
		t.Error("different requirement descriptions should produce different hashes")
	}

	// Test with multiple features
	s5 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
		Features: []spec.Feature{
			{ID: "f1", Description: "Feature 1"},
			{ID: "f2", Description: "Feature 2"},
		},
	}

	s6 := &spec.ProductSpec{
		ID:      "test-1",
		Version: "1.0",
		Title:   "Test Spec",
		Features: []spec.Feature{
			{ID: "f1", Description: "Feature 1"},
		},
	}

	if s5.Hash() == s6.Hash() {
		t.Error("specs with different feature count should produce different hashes")
	}
}

func TestSource_StringAndIsZero(t *testing.T) {
	cases := []struct {
		name    string
		src     spec.Source
		isZero  bool
		want    string
	}{
		{"zero", spec.Source{}, true, ""},
		{"doc only", spec.Source{Doc: "docs/x.md"}, false, "docs/x.md"},
		{"doc and line", spec.Source{Doc: "docs/x.md", Line: 7}, false, "docs/x.md:7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.src.IsZero(); got != tc.isZero {
				t.Errorf("IsZero() = %v, want %v", got, tc.isZero)
			}
			if got := tc.src.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
