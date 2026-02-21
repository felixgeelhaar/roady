package application

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type BillingService struct {
	repo  domain.WorkspaceRepository
	audit domain.AuditLogger
}

func NewBillingService(repo domain.WorkspaceRepository, audit domain.AuditLogger) *BillingService {
	return &BillingService{
		repo:  repo,
		audit: audit,
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
		result := state.TaskStates[taskID]
		result.RateID = rateID
		state.TaskStates[taskID] = result
		state.UpdatedAt = time.Now()
	}

	if err := s.repo.SaveState(state); err != nil {
		return err
	}

	_ = s.audit.Log("billing.task_started", "system", map[string]interface{}{
		"task_id": taskID,
		"rate_id": rateID,
	})

	return nil
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

	// Re-read the result after CompleteTask to get elapsed minutes
	result = state.TaskStates[taskID]

	_ = s.audit.Log("billing.task_completed", "system", map[string]interface{}{
		"task_id":         taskID,
		"elapsed_minutes": result.ElapsedMinutes,
	})

	return s.createTimeEntryFromTask(taskID, &result, state)
}

func (s *BillingService) LogTime(taskID string, rateID string, minutes int, description string) error {
	if rateID == "" {
		rateID = s.getDefaultRateID()
	}
	if rateID == "" {
		return fmt.Errorf("no rate specified and no default rate configured")
	}

	entryID := generateTimeEntryID()
	entry, err := billing.NewTimeEntry(entryID, taskID, rateID, minutes, description, time.Now())
	if err != nil {
		return err
	}

	entries, err := s.repo.LoadTimeEntries()
	if err != nil {
		return fmt.Errorf("failed to load time entries: %w", err)
	}

	entries = append(entries, entry)
	if err := s.repo.SaveTimeEntries(entries); err != nil {
		return err
	}

	_ = s.audit.Log("billing.time_logged", "system", map[string]interface{}{
		"task_id": taskID,
		"rate_id": rateID,
		"minutes": minutes,
	})

	return nil
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
	taskEstimates := make(map[string]float64)
	if plan != nil {
		for _, task := range plan.Tasks {
			taskTitles[task.ID] = task.Title
			if task.Estimate != "" {
				est, err := planning.ParseEstimate(task.Estimate)
				if err == nil && !est.IsZero() {
					taskEstimates[task.ID] = est.Hours()
				}
			}
		}
	}

	entries, err := s.repo.LoadTimeEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to load time entries: %w", err)
	}

	// Track tasks that already have time entries to avoid double-counting
	// with elapsed minutes still present in task state.
	tasksWithEntries := make(map[string]bool)

	for _, entry := range entries {
		if opts.TaskID != "" && entry.TaskID != opts.TaskID {
			continue
		}

		tasksWithEntries[entry.TaskID] = true

		rate := config.GetByID(entry.RateID)
		if rate == nil {
			continue
		}

		hours := entry.Hours()
		cost := hours * rate.HourlyRate
		estHours := taskEstimates[entry.TaskID]
		estCost := estHours * rate.HourlyRate

		report.AddEntry(billing.CostReportEntry{
			TaskID:         entry.TaskID,
			Title:          taskTitles[entry.TaskID],
			RateID:         entry.RateID,
			RateName:       rate.Name,
			Hours:          hours,
			HourlyRate:     rate.HourlyRate,
			Cost:           cost,
			Description:    entry.Description,
			EstimatedHours: estHours,
			EstimatedCost:  estCost,
			CostVariance:   cost - estCost,
			HoursVariance:  hours - estHours,
		}, config.Tax)
	}

	for taskID, result := range state.TaskStates {
		if opts.TaskID != "" && taskID != opts.TaskID {
			continue
		}

		if result.ElapsedMinutes == 0 || tasksWithEntries[taskID] {
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

		hours := float64(result.ElapsedMinutes) / 60.0
		cost := hours * rate.HourlyRate
		estHours := taskEstimates[taskID]
		estCost := estHours * rate.HourlyRate

		report.AddEntry(billing.CostReportEntry{
			TaskID:         taskID,
			Title:          taskTitles[taskID],
			RateID:         rateID,
			RateName:       rate.Name,
			Hours:          hours,
			HourlyRate:     rate.HourlyRate,
			Cost:           cost,
			EstimatedHours: estHours,
			EstimatedCost:  estCost,
			CostVariance:   cost - estCost,
			HoursVariance:  hours - estHours,
		}, config.Tax)
	}

	report.ComputeCoverage()

	return report, nil
}

func (s *BillingService) AddRate(rate billing.Rate) error {
	config, err := s.repo.LoadRates()
	if err != nil {
		return fmt.Errorf("failed to load rates: %w", err)
	}

	if err := config.AddRate(rate); err != nil {
		return err
	}

	if err := s.repo.SaveRates(config); err != nil {
		return err
	}

	_ = s.audit.Log("billing.rate_added", "system", map[string]interface{}{
		"rate_id":     rate.ID,
		"rate_name":   rate.Name,
		"hourly_rate": rate.HourlyRate,
	})

	return nil
}

func (s *BillingService) RemoveRate(rateID string) error {
	config, err := s.repo.LoadRates()
	if err != nil {
		return fmt.Errorf("failed to load rates: %w", err)
	}

	if err := config.RemoveRate(rateID); err != nil {
		return err
	}

	if err := s.repo.SaveRates(config); err != nil {
		return err
	}

	_ = s.audit.Log("billing.rate_removed", "system", map[string]interface{}{
		"rate_id": rateID,
	})

	return nil
}

func (s *BillingService) ListRates() (*billing.RateConfig, error) {
	return s.repo.LoadRates()
}

func (s *BillingService) SetDefaultRate(rateID string) error {
	config, err := s.repo.LoadRates()
	if err != nil {
		return fmt.Errorf("failed to load rates: %w", err)
	}

	if err := config.SetDefault(rateID); err != nil {
		return err
	}

	if err := s.repo.SaveRates(config); err != nil {
		return err
	}

	_ = s.audit.Log("billing.default_rate_set", "system", map[string]interface{}{
		"rate_id": rateID,
	})

	return nil
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

	if err := s.repo.SaveRates(config); err != nil {
		return err
	}

	_ = s.audit.Log("billing.tax_configured", "system", map[string]interface{}{
		"tax_name":    name,
		"tax_percent": percent,
		"included":    included,
	})

	return nil
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

	entryID := generateTimeEntryID()
	entry, err := billing.NewTimeEntry(entryID, taskID, rateID, result.ElapsedMinutes, "", time.Now())
	if err != nil {
		return err
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

func (s *BillingService) GetBudgetStatus() (*billing.BudgetStatus, error) {
	policy, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	if policy.BudgetHours == 0 {
		return nil, nil
	}

	totalMinutes := s.getTotalMinutes()

	plan, err := s.repo.LoadPlan()
	if err != nil || plan == nil {
		return billing.NewBudgetStatus(policy.BudgetHours, totalMinutes), nil
	}

	config, err := s.repo.LoadRates()
	if err != nil {
		return billing.NewBudgetStatus(policy.BudgetHours, totalMinutes), nil
	}

	defaultRate := config.GetDefault()
	if defaultRate == nil {
		return billing.NewBudgetStatus(policy.BudgetHours, totalMinutes), nil
	}

	var estimatedHours float64
	var estimatedTasks int
	for _, task := range plan.Tasks {
		if task.Estimate != "" {
			est, err := planning.ParseEstimate(task.Estimate)
			if err == nil && !est.IsZero() {
				estimatedHours += est.Hours()
				estimatedTasks++
			}
		}
	}

	currency := config.Currency
	if currency == "" {
		currency = "USD"
	}

	return billing.NewBudgetStatusWithEstimates(
		policy.BudgetHours, totalMinutes,
		estimatedHours, defaultRate.HourlyRate,
		len(plan.Tasks), estimatedTasks,
		currency,
	), nil
}

func (s *BillingService) getTotalMinutes() int {
	totalMinutes := 0
	tasksWithEntries := make(map[string]bool)

	entries, err := s.repo.LoadTimeEntries()
	if err == nil {
		for _, entry := range entries {
			totalMinutes += entry.Minutes
			tasksWithEntries[entry.TaskID] = true
		}
	}

	state, err := s.repo.LoadState()
	if err == nil {
		for taskID, result := range state.TaskStates {
			if !tasksWithEntries[taskID] {
				totalMinutes += result.ElapsedMinutes
			}
		}
	}

	return totalMinutes
}
