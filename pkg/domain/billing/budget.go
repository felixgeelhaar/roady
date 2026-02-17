package billing

// BudgetStatus represents the current budget consumption state.
type BudgetStatus struct {
	BudgetHours      int     `json:"budget_hours"`
	UsedHours        float64 `json:"used_hours"`
	Remaining        float64 `json:"remaining"`
	PercentUsed      float64 `json:"percent_used"`
	OverBudget       bool    `json:"over_budget"`
	EstimatedHours   float64 `json:"estimated_hours"`
	EstimatedCost    float64 `json:"estimated_cost"`
	ActualCost       float64 `json:"actual_cost"`
	CostVariance     float64 `json:"cost_variance"`
	EstimateCoverage float64 `json:"estimate_coverage"`
	UnestimatedTasks int     `json:"unestimated_tasks"`
	HourlyRate       float64 `json:"hourly_rate"`
	Currency         string  `json:"currency"`
}

// NewBudgetStatus creates a BudgetStatus from budget hours and total consumed minutes.
func NewBudgetStatus(budgetHours, totalMinutes int) *BudgetStatus {
	usedHours := float64(totalMinutes) / 60.0
	remaining := float64(budgetHours) - usedHours
	percentUsed := (usedHours / float64(budgetHours)) * 100

	return &BudgetStatus{
		BudgetHours: budgetHours,
		UsedHours:   usedHours,
		Remaining:   remaining,
		PercentUsed: percentUsed,
		OverBudget:  remaining < 0,
	}
}

// NewBudgetStatusWithEstimates creates a BudgetStatus enriched with estimated cost data.
func NewBudgetStatusWithEstimates(budgetHours, totalMinutes int, estimatedHours, hourlyRate float64, totalTasks, estimatedTasks int, currency string) *BudgetStatus {
	status := NewBudgetStatus(budgetHours, totalMinutes)

	status.EstimatedHours = estimatedHours
	status.EstimatedCost = estimatedHours * hourlyRate
	status.ActualCost = status.UsedHours * hourlyRate
	status.CostVariance = status.ActualCost - status.EstimatedCost
	status.HourlyRate = hourlyRate
	status.Currency = currency

	if totalTasks > 0 {
		status.EstimateCoverage = (float64(estimatedTasks) / float64(totalTasks)) * 100
	}
	status.UnestimatedTasks = totalTasks - estimatedTasks

	return status
}
