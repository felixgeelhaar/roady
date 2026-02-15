package application

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type BillingService struct {
	repo domain.WorkspaceRepository
}

func NewBillingService(repo domain.WorkspaceRepository) *BillingService {
	return &BillingService{
		repo: repo,
	}
}

type CostReportOpts struct {
	TaskID string
	Period string
	Format string
}

func (s *BillingService) StartTask(taskID string, rateID string) error {
	state, err := s.repo.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	currentStatus := state.GetTaskStatus(taskID)
	if currentStatus == planning.StatusInProgress {
		return fmt.Errorf("task %s is already in progress", taskID)
	}

	state.StartTask(taskID)

	if rateID == "" {
		rateID = s.getDefaultRateID()
	}
	if rateID != "" {
		state.SetTaskRate(taskID, rateID)
	}

	return s.repo.SaveState(state)
}

func (s *BillingService) CompleteTask(taskID string) error {
	state, err := s.repo.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	result, ok := state.GetTaskResult(taskID)
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	if result.Status != planning.StatusInProgress {
		return fmt.Errorf("task %s is not in progress", taskID)
	}

	state.CompleteTask(taskID)

	if err := s.repo.SaveState(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return s.createTimeEntryFromTask(taskID, &result, state)
}

func (s *BillingService) LogTime(taskID string, rateID string, minutes int, description string) error {
	if minutes <= 0 {
		return fmt.Errorf("minutes must be positive")
	}

	if rateID == "" {
		rateID = s.getDefaultRateID()
	}
	if rateID == "" {
		return fmt.Errorf("no rate specified and no default rate configured")
	}

	entry := billing.TimeEntry{
		ID:          generateTimeEntryID(),
		TaskID:      taskID,
		RateID:      rateID,
		Minutes:     minutes,
		Description: description,
		CreatedAt:   time.Now(),
	}

	entries, err := s.repo.LoadTimeEntries()
	if err != nil {
		return fmt.Errorf("failed to load time entries: %w", err)
	}

	entries = append(entries, entry)
	return s.repo.SaveTimeEntries(entries)
}

func (s *BillingService) GetRate(rateID string) (*billing.Rate, error) {
	config, err := s.repo.LoadRates()
	if err != nil {
		return nil, fmt.Errorf("failed to load rates: %w", err)
	}

	rate := config.GetByID(rateID)
	if rate == nil {
		return nil, fmt.Errorf("rate %s not found", rateID)
	}

	return rate, nil
}

func (s *BillingService) GetDefaultRate() (*billing.Rate, error) {
	config, err := s.repo.LoadRates()
	if err != nil {
		return nil, fmt.Errorf("failed to load rates: %w", err)
	}

	rate := config.GetDefault()
	if rate == nil {
		return nil, nil
	}

	return rate, nil
}

func (s *BillingService) GetCostReport(opts CostReportOpts) (*billing.CostReport, error) {
	config, err := s.repo.LoadRates()
	if err != nil {
		return nil, fmt.Errorf("failed to load rates: %w", err)
	}

	currency := config.Currency
	if currency == "" {
		currency = "USD"
	}

	report := billing.NewCostReport(currency)
	report.SetTax(config.Tax)

	state, err := s.repo.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	plan, err := s.repo.LoadPlan()
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	taskTitles := make(map[string]string)
	if plan != nil {
		for _, task := range plan.Tasks {
			taskTitles[task.ID] = task.Title
		}
	}

	entries, err := s.repo.LoadTimeEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to load time entries: %w", err)
	}

	for _, entry := range entries {
		if opts.TaskID != "" && entry.TaskID != opts.TaskID {
			continue
		}

		rate := config.GetByID(entry.RateID)
		if rate == nil {
			continue
		}

		hours := entry.Hours()
		cost := hours * rate.HourlyRate

		report.AddEntry(billing.CostReportEntry{
			TaskID:      entry.TaskID,
			Title:       taskTitles[entry.TaskID],
			RateID:      entry.RateID,
			RateName:    rate.Name,
			Hours:       hours,
			HourlyRate:  rate.HourlyRate,
			Cost:        cost,
			Description: entry.Description,
		}, config.Tax)
	}

	for taskID, result := range state.TaskStates {
		if opts.TaskID != "" && taskID != opts.TaskID {
			continue
		}

		if result.ElapsedMinutes == 0 {
			continue
		}

		rateID := result.RateID
		if rateID == "" {
			rateID = s.getDefaultRateID()
		}
		if rateID == "" {
			continue
		}

		rate := config.GetByID(rateID)
		if rate == nil {
			continue
		}

		hours := result.GetElapsedHours()
		cost := hours * rate.HourlyRate

		report.AddEntry(billing.CostReportEntry{
			TaskID:     taskID,
			Title:      taskTitles[taskID],
			RateID:     rateID,
			RateName:   rate.Name,
			Hours:      hours,
			HourlyRate: rate.HourlyRate,
			Cost:       cost,
		}, config.Tax)
	}

	return report, nil
}

func (s *BillingService) AddRate(rate billing.Rate) error {
	config, err := s.repo.LoadRates()
	if err != nil {
		return fmt.Errorf("failed to load rates: %w", err)
	}

	if rate.IsDefault {
		for i := range config.Rates {
			config.Rates[i].IsDefault = false
		}
	}

	for _, r := range config.Rates {
		if r.ID == rate.ID {
			return fmt.Errorf("rate %s already exists", rate.ID)
		}
	}

	config.Rates = append(config.Rates, rate)
	return s.repo.SaveRates(config)
}

func (s *BillingService) RemoveRate(rateID string) error {
	config, err := s.repo.LoadRates()
	if err != nil {
		return fmt.Errorf("failed to load rates: %w", err)
	}

	found := false
	newRates := []billing.Rate{}
	for _, r := range config.Rates {
		if r.ID == rateID {
			found = true
			continue
		}
		newRates = append(newRates, r)
	}

	if !found {
		return fmt.Errorf("rate %s not found", rateID)
	}

	config.Rates = newRates
	return s.repo.SaveRates(config)
}

func (s *BillingService) ListRates() (*billing.RateConfig, error) {
	return s.repo.LoadRates()
}

func (s *BillingService) SetDefaultRate(rateID string) error {
	config, err := s.repo.LoadRates()
	if err != nil {
		return fmt.Errorf("failed to load rates: %w", err)
	}

	found := false
	for i, r := range config.Rates {
		if r.ID == rateID {
			found = true
			config.Rates[i].IsDefault = true
		} else {
			config.Rates[i].IsDefault = false
		}
	}

	if !found {
		return fmt.Errorf("rate %s not found", rateID)
	}

	return s.repo.SaveRates(config)
}

func (s *BillingService) SetTax(name string, percent float64, included bool) error {
	config, err := s.repo.LoadRates()
	if err != nil {
		return fmt.Errorf("failed to load rates: %w", err)
	}

	config.Tax = &billing.TaxConfig{
		Name:     name,
		Percent:  percent,
		Included: included,
	}

	return s.repo.SaveRates(config)
}

func (s *BillingService) createTimeEntryFromTask(taskID string, result *planning.TaskResult, state *planning.ExecutionState) error {
	if result.ElapsedMinutes == 0 {
		return nil
	}

	rateID := result.RateID
	if rateID == "" {
		rateID = s.getDefaultRateID()
	}
	if rateID == "" {
		return nil
	}

	entry := billing.TimeEntry{
		ID:          generateTimeEntryID(),
		TaskID:      taskID,
		RateID:      rateID,
		Minutes:     result.ElapsedMinutes,
		Description: "",
		CreatedAt:   time.Now(),
	}

	entries, err := s.repo.LoadTimeEntries()
	if err != nil {
		return fmt.Errorf("failed to load time entries: %w", err)
	}

	entries = append(entries, entry)
	return s.repo.SaveTimeEntries(entries)
}

func (s *BillingService) getDefaultRateID() string {
	config, err := s.repo.LoadRates()
	if err != nil {
		return ""
	}

	rate := config.GetDefault()
	if rate == nil {
		return ""
	}

	return rate.ID
}

func generateTimeEntryID() string {
	return fmt.Sprintf("te-%d", time.Now().UnixNano())
}

type BudgetStatus struct {
	BudgetHours int     `json:"budget_hours"`
	UsedHours   float64 `json:"used_hours"`
	Remaining   float64 `json:"remaining"`
	PercentUsed float64 `json:"percent_used"`
	OverBudget  bool    `json:"over_budget"`
}

func (s *BillingService) GetBudgetStatus() (*BudgetStatus, error) {
	policy, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	if policy.BudgetHours == 0 {
		return nil, nil
	}

	totalMinutes := s.getTotalMinutes()
	usedHours := float64(totalMinutes) / 60.0
	remaining := float64(policy.BudgetHours) - usedHours
	percentUsed := (usedHours / float64(policy.BudgetHours)) * 100

	return &BudgetStatus{
		BudgetHours: policy.BudgetHours,
		UsedHours:   usedHours,
		Remaining:   remaining,
		PercentUsed: percentUsed,
		OverBudget:  remaining < 0,
	}, nil
}

func (s *BillingService) getTotalMinutes() int {
	totalMinutes := 0

	entries, err := s.repo.LoadTimeEntries()
	if err == nil {
		for _, entry := range entries {
			totalMinutes += entry.Minutes
		}
	}

	state, err := s.repo.LoadState()
	if err == nil {
		for _, result := range state.TaskStates {
			totalMinutes += result.ElapsedMinutes
		}
	}

	return totalMinutes
}
