package drift_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/internal/domain/drift"
)

func TestReport_HasCriticalDrift(t *testing.T) {
	tests := []struct {
		name string
		issues []drift.Issue
		want bool
	}{
		{
			name: "No Issues",
			issues: []drift.Issue{},
			want: false,
		},
		{
			name: "Low Severity",
			issues: []drift.Issue{{Severity: drift.SeverityLow}},
			want: false,
		},
		{
			name: "Critical Severity",
			issues: []drift.Issue{{Severity: drift.SeverityCritical}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &drift.Report{Issues: tt.issues}
			if got := r.HasCriticalDrift(); got != tt.want {
				t.Errorf("HasCriticalDrift() = %v, want %v", got, tt.want)
			}
		})
	}
}
