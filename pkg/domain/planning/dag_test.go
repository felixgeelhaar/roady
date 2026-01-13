package planning_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestPlan_ValidateDAG(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []planning.Task
		wantErr bool
	}{
		{
			name: "No Dependencies",
			tasks: []planning.Task{
				{ID: "A"}, {ID: "B"},
			},
			wantErr: false,
		},
		{
			name: "Linear Chain",
			tasks: []planning.Task{
				{ID: "A", DependsOn: []string{"B"}},
				{ID: "B", DependsOn: []string{"C"}},
				{ID: "C"},
			},
			wantErr: false,
		},
		{
			name: "Simple Cycle",
			tasks: []planning.Task{
				{ID: "A", DependsOn: []string{"B"}},
				{ID: "B", DependsOn: []string{"A"}},
			},
			wantErr: true,
		},
		{
			name: "Self Reference",
			tasks: []planning.Task{
				{ID: "A", DependsOn: []string{"A"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &planning.Plan{Tasks: tt.tasks}
			err := p.ValidateDAG()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDAG() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
