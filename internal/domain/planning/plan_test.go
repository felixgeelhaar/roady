package planning_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

func TestPlan_Hash(t *testing.T) {
	p1 := &planning.Plan{ID: "1", SpecID: "s1", Tasks: []planning.Task{{ID: "t1"}}}
	p2 := &planning.Plan{ID: "1", SpecID: "s1", Tasks: []planning.Task{{ID: "t1"}}}
	p3 := &planning.Plan{ID: "2", SpecID: "s1", Tasks: []planning.Task{{ID: "t1"}}}

	if p1.Hash() != p2.Hash() {
		t.Error("Hashes should match")
	}
	if p1.Hash() == p3.Hash() {
		t.Error("Hashes should differ")
	}
}
