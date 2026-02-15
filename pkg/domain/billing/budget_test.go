package billing

import "testing"

func TestNewBudgetStatus(t *testing.T) {
	tests := []struct {
		name         string
		budgetHours  int
		totalMinutes int
		wantUsed     float64
		wantRemain   float64
		wantOver     bool
	}{
		{
			name:         "normal usage",
			budgetHours:  100,
			totalMinutes: 600, // 10 hours
			wantUsed:     10.0,
			wantRemain:   90.0,
			wantOver:     false,
		},
		{
			name:         "exactly at budget",
			budgetHours:  10,
			totalMinutes: 600, // 10 hours
			wantUsed:     10.0,
			wantRemain:   0.0,
			wantOver:     false,
		},
		{
			name:         "over budget",
			budgetHours:  10,
			totalMinutes: 720, // 12 hours
			wantUsed:     12.0,
			wantRemain:   -2.0,
			wantOver:     true,
		},
		{
			name:         "no usage",
			budgetHours:  100,
			totalMinutes: 0,
			wantUsed:     0.0,
			wantRemain:   100.0,
			wantOver:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewBudgetStatus(tt.budgetHours, tt.totalMinutes)
			if status.UsedHours != tt.wantUsed {
				t.Errorf("UsedHours: want %f, got %f", tt.wantUsed, status.UsedHours)
			}
			if status.Remaining != tt.wantRemain {
				t.Errorf("Remaining: want %f, got %f", tt.wantRemain, status.Remaining)
			}
			if status.OverBudget != tt.wantOver {
				t.Errorf("OverBudget: want %v, got %v", tt.wantOver, status.OverBudget)
			}
			if status.BudgetHours != tt.budgetHours {
				t.Errorf("BudgetHours: want %d, got %d", tt.budgetHours, status.BudgetHours)
			}
		})
	}
}
