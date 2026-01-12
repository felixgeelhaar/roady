package spec_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/internal/domain/spec"
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
