package spec_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestProductSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    spec.ProductSpec
		wantErr bool
	}{
		{
			name: "Valid Spec",
			spec: spec.ProductSpec{ID: "s1", Title: "T1", Features: []spec.Feature{{ID: "f1"}}},
			wantErr: false,
		},
		{
			name: "Missing ID",
			spec: spec.ProductSpec{Title: "T1", Features: []spec.Feature{{ID: "f1"}}},
			wantErr: true,
		},
		{
			name: "Missing Title",
			spec: spec.ProductSpec{ID: "s1", Features: []spec.Feature{{ID: "f1"}}},
			wantErr: true,
		},
		{
			name: "No Features",
			spec: spec.ProductSpec{ID: "s1", Title: "T1"},
			wantErr: true,
		},
		{
			name: "Duplicate Feature ID",
			spec: spec.ProductSpec{ID: "s1", Title: "T1", Features: []spec.Feature{{ID: "f1"}, {ID: "f1"}}},
			wantErr: true,
		},
		{
			name: "Missing Feature ID",
			spec: spec.ProductSpec{ID: "s1", Title: "T1", Features: []spec.Feature{{ID: ""}}},
			wantErr: true,
		},
		{
			name: "Missing Requirement ID",
			spec: spec.ProductSpec{ID: "s1", Title: "T1", Features: []spec.Feature{{ID: "f1", Requirements: []spec.Requirement{{ID: "", Title: "R1"}}}}},
			wantErr: true,
		},
		{
			name: "Missing Requirement Title",
			spec: spec.ProductSpec{ID: "s1", Title: "T1", Features: []spec.Feature{{ID: "f1", Requirements: []spec.Requirement{{ID: "r1", Title: ""}}}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.spec.Validate()
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() has errors: %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}
