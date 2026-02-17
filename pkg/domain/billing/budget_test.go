package billing

import (
	"math"
	"testing"
)

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

func TestNewBudgetStatusWithEstimates(t *testing.T) {
	tests := []struct {
		name              string
		budgetHours       int
		totalMinutes      int
		estimatedHours    float64
		hourlyRate        float64
		totalTasks        int
		estimatedTasks    int
		currency          string
		wantEstCost       float64
		wantActualCost    float64
		wantVariance      float64
		wantCoverage      float64
		wantUnestimated   int
	}{
		{
			name:            "full estimates, under budget",
			budgetHours:     100,
			totalMinutes:    600, // 10 hours
			estimatedHours:  20.0,
			hourlyRate:      150.0,
			totalTasks:      5,
			estimatedTasks:  5,
			currency:        "USD",
			wantEstCost:     3000.0,
			wantActualCost:  1500.0,
			wantVariance:    -1500.0,
			wantCoverage:    100.0,
			wantUnestimated: 0,
		},
		{
			name:            "partial estimates",
			budgetHours:     100,
			totalMinutes:    300, // 5 hours
			estimatedHours:  8.0,
			hourlyRate:      100.0,
			totalTasks:      4,
			estimatedTasks:  2,
			currency:        "EUR",
			wantEstCost:     800.0,
			wantActualCost:  500.0,
			wantVariance:    -300.0,
			wantCoverage:    50.0,
			wantUnestimated: 2,
		},
		{
			name:            "no estimates",
			budgetHours:     100,
			totalMinutes:    120, // 2 hours
			estimatedHours:  0,
			hourlyRate:      100.0,
			totalTasks:      3,
			estimatedTasks:  0,
			currency:        "USD",
			wantEstCost:     0,
			wantActualCost:  200.0,
			wantVariance:    200.0,
			wantCoverage:    0,
			wantUnestimated: 3,
		},
		{
			name:            "over estimate",
			budgetHours:     100,
			totalMinutes:    600, // 10 hours
			estimatedHours:  5.0,
			hourlyRate:      100.0,
			totalTasks:      2,
			estimatedTasks:  2,
			currency:        "USD",
			wantEstCost:     500.0,
			wantActualCost:  1000.0,
			wantVariance:    500.0,
			wantCoverage:    100.0,
			wantUnestimated: 0,
		},
		{
			name:            "zero tasks",
			budgetHours:     100,
			totalMinutes:    0,
			estimatedHours:  0,
			hourlyRate:      100.0,
			totalTasks:      0,
			estimatedTasks:  0,
			currency:        "USD",
			wantEstCost:     0,
			wantActualCost:  0,
			wantVariance:    0,
			wantCoverage:    0,
			wantUnestimated: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewBudgetStatusWithEstimates(
				tt.budgetHours, tt.totalMinutes,
				tt.estimatedHours, tt.hourlyRate,
				tt.totalTasks, tt.estimatedTasks,
				tt.currency,
			)

			// Base fields should still work
			wantUsed := float64(tt.totalMinutes) / 60.0
			if status.UsedHours != wantUsed {
				t.Errorf("UsedHours: want %f, got %f", wantUsed, status.UsedHours)
			}

			if status.EstimatedHours != tt.estimatedHours {
				t.Errorf("EstimatedHours: want %f, got %f", tt.estimatedHours, status.EstimatedHours)
			}
			if math.Abs(status.EstimatedCost-tt.wantEstCost) > 0.01 {
				t.Errorf("EstimatedCost: want %f, got %f", tt.wantEstCost, status.EstimatedCost)
			}
			if math.Abs(status.ActualCost-tt.wantActualCost) > 0.01 {
				t.Errorf("ActualCost: want %f, got %f", tt.wantActualCost, status.ActualCost)
			}
			if math.Abs(status.CostVariance-tt.wantVariance) > 0.01 {
				t.Errorf("CostVariance: want %f, got %f", tt.wantVariance, status.CostVariance)
			}
			if math.Abs(status.EstimateCoverage-tt.wantCoverage) > 0.01 {
				t.Errorf("EstimateCoverage: want %f, got %f", tt.wantCoverage, status.EstimateCoverage)
			}
			if status.UnestimatedTasks != tt.wantUnestimated {
				t.Errorf("UnestimatedTasks: want %d, got %d", tt.wantUnestimated, status.UnestimatedTasks)
			}
			if status.HourlyRate != tt.hourlyRate {
				t.Errorf("HourlyRate: want %f, got %f", tt.hourlyRate, status.HourlyRate)
			}
			if status.Currency != tt.currency {
				t.Errorf("Currency: want %s, got %s", tt.currency, status.Currency)
			}
		})
	}
}
