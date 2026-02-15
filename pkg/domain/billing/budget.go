package billing

// BudgetStatus represents the current budget consumption state.
type BudgetStatus struct {
	BudgetHours int     `json:"budget_hours"`
	UsedHours   float64 `json:"used_hours"`
	Remaining   float64 `json:"remaining"`
	PercentUsed float64 `json:"percent_used"`
	OverBudget  bool    `json:"over_budget"`
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
