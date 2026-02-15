package application_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type mockBillingRepo struct {
	*MockRepo
	RatesConfig *billing.RateConfig
	TimeEntries []billing.TimeEntry
	ExecState   *planning.ExecutionState
	Plan        *planning.Plan
	Policy      *domain.PolicyConfig
}

func (m *mockBillingRepo) LoadRates() (*billing.RateConfig, error) {
	if m.RatesConfig == nil {
		return &billing.RateConfig{}, nil
	}
	return m.RatesConfig, nil
}

func (m *mockBillingRepo) SaveRates(c *billing.RateConfig) error {
	m.RatesConfig = c
	return nil
}

func (m *mockBillingRepo) LoadTimeEntries() ([]billing.TimeEntry, error) {
	return m.TimeEntries, nil
}

func (m *mockBillingRepo) SaveTimeEntries(e []billing.TimeEntry) error {
	m.TimeEntries = e
	return nil
}

func (m *mockBillingRepo) LoadState() (*planning.ExecutionState, error) {
	if m.ExecState == nil {
		return planning.NewExecutionState("test"), nil
	}
	return m.ExecState, nil
}

func (m *mockBillingRepo) SaveState(s *planning.ExecutionState) error {
	m.ExecState = s
	return nil
}

func (m *mockBillingRepo) LoadPlan() (*planning.Plan, error) {
	return m.Plan, nil
}

func (m *mockBillingRepo) LoadPolicy() (*domain.PolicyConfig, error) {
	if m.Policy == nil {
		return &domain.PolicyConfig{}, nil
	}
	return m.Policy, nil
}

func TestBillingService_GetRate(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100},
			},
		},
	}

	svc := application.NewBillingService(repo)

	rate, err := svc.GetRate("rate-1")
	if err != nil {
		t.Fatalf("GetRate failed: %v", err)
	}
	if rate == nil {
		t.Fatal("expected rate, got nil")
	}
	if rate.ID != "rate-1" {
		t.Errorf("expected rate-1, got %s", rate.ID)
	}

	_, err = svc.GetRate("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rate")
	}
}

func TestBillingService_GetDefaultRate(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100, IsDefault: true},
			},
		},
	}

	svc := application.NewBillingService(repo)

	rate, err := svc.GetDefaultRate()
	if err != nil {
		t.Fatalf("GetDefaultRate failed: %v", err)
	}
	if rate == nil {
		t.Fatal("expected rate, got nil")
	}
	if !rate.IsDefault {
		t.Error("expected default rate")
	}
}

func TestBillingService_GetDefaultRate_NoDefault(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo:    &MockRepo{},
		RatesConfig: &billing.RateConfig{Rates: []billing.Rate{}},
	}

	svc := application.NewBillingService(repo)

	rate, err := svc.GetDefaultRate()
	if err != nil {
		t.Fatalf("GetDefaultRate failed: %v", err)
	}
	if rate != nil {
		t.Error("expected nil for no default rate")
	}
}

func TestBillingService_AddRate(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{},
		},
	}

	svc := application.NewBillingService(repo)

	err := svc.AddRate(billing.Rate{ID: "rate-1", Name: "Standard", HourlyRate: 100})
	if err != nil {
		t.Fatalf("AddRate failed: %v", err)
	}

	config, _ := repo.LoadRates()
	if len(config.Rates) != 1 {
		t.Errorf("expected 1 rate, got %d", len(config.Rates))
	}
}

func TestBillingService_AddRate_Duplicate(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100},
			},
		},
	}

	svc := application.NewBillingService(repo)

	err := svc.AddRate(billing.Rate{ID: "rate-1", Name: "Standard", HourlyRate: 100})
	if err == nil {
		t.Fatal("expected error for duplicate rate")
	}
}

func TestBillingService_AddRate_SetDefault(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100, IsDefault: true},
			},
		},
	}

	svc := application.NewBillingService(repo)

	err := svc.AddRate(billing.Rate{ID: "rate-2", Name: "Premium", HourlyRate: 200, IsDefault: true})
	if err != nil {
		t.Fatalf("AddRate failed: %v", err)
	}

	config, _ := repo.LoadRates()
	if len(config.Rates) != 2 {
		t.Errorf("expected 2 rates, got %d", len(config.Rates))
	}

	var defaultCount int
	for _, r := range config.Rates {
		if r.IsDefault {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		t.Errorf("expected 1 default rate, got %d", defaultCount)
	}
}

func TestBillingService_RemoveRate(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100},
			},
		},
	}

	svc := application.NewBillingService(repo)

	err := svc.RemoveRate("rate-1")
	if err != nil {
		t.Fatalf("RemoveRate failed: %v", err)
	}

	config, _ := repo.LoadRates()
	if len(config.Rates) != 0 {
		t.Errorf("expected 0 rates, got %d", len(config.Rates))
	}
}

func TestBillingService_RemoveRate_NotFound(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo:    &MockRepo{},
		RatesConfig: &billing.RateConfig{},
	}

	svc := application.NewBillingService(repo)

	err := svc.RemoveRate("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rate")
	}
}

func TestBillingService_SetDefaultRate(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100},
				{ID: "rate-2", Name: "Premium", HourlyRate: 200},
			},
		},
	}

	svc := application.NewBillingService(repo)

	err := svc.SetDefaultRate("rate-2")
	if err != nil {
		t.Fatalf("SetDefaultRate failed: %v", err)
	}

	config, _ := repo.LoadRates()
	for _, r := range config.Rates {
		if r.ID == "rate-2" && !r.IsDefault {
			t.Error("rate-2 should be default")
		}
		if r.ID == "rate-1" && r.IsDefault {
			t.Error("rate-1 should not be default")
		}
	}
}

func TestBillingService_SetTax(t *testing.T) {
	repo := &mockBillingRepo{MockRepo: &MockRepo{}}

	svc := application.NewBillingService(repo)

	err := svc.SetTax("VAT", 20.0, false)
	if err != nil {
		t.Fatalf("SetTax failed: %v", err)
	}

	config, _ := repo.LoadRates()
	if config.Tax == nil {
		t.Fatal("expected tax config")
	}
	if config.Tax.Name != "VAT" {
		t.Errorf("expected VAT, got %s", config.Tax.Name)
	}
	if config.Tax.Percent != 20.0 {
		t.Errorf("expected 20.0, got %f", config.Tax.Percent)
	}
}

func TestBillingService_GetCostReport(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Currency: "USD",
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100},
			},
		},
		TimeEntries: []billing.TimeEntry{
			{TaskID: "task-1", RateID: "rate-1", Minutes: 60},
		},
		ExecState: func() *planning.ExecutionState {
			s := planning.NewExecutionState("test")
			s.TaskStates["task-2"] = planning.TaskResult{
				RateID:         "rate-1",
				ElapsedMinutes: 120,
			}
			return s
		}(),
		Plan: &planning.Plan{
			Tasks: []planning.Task{
				{ID: "task-1", Title: "Task 1"},
				{ID: "task-2", Title: "Task 2"},
			},
		},
	}

	svc := application.NewBillingService(repo)

	report, err := svc.GetCostReport(application.CostReportOpts{})
	if err != nil {
		t.Fatalf("GetCostReport failed: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if report.TotalHours != 3.0 {
		t.Errorf("expected 3.0 hours, got %f", report.TotalHours)
	}
}

func TestBillingService_GetCostReport_WithTax(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Currency: "USD",
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100},
			},
			Tax: &billing.TaxConfig{Name: "VAT", Percent: 20, Included: false},
		},
		TimeEntries: []billing.TimeEntry{
			{TaskID: "task-1", RateID: "rate-1", Minutes: 60},
		},
	}

	svc := application.NewBillingService(repo)

	report, err := svc.GetCostReport(application.CostReportOpts{})
	if err != nil {
		t.Fatalf("GetCostReport failed: %v", err)
	}
	if report.TaxPercent != 20 {
		t.Errorf("expected 20%% tax, got %f", report.TaxPercent)
	}
}

func TestBillingService_GetBudgetStatus(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		Policy:   &domain.PolicyConfig{BudgetHours: 100},
		TimeEntries: []billing.TimeEntry{
			{Minutes: 600},
		},
		ExecState: func() *planning.ExecutionState {
			s := planning.NewExecutionState("test")
			s.TaskStates["task-1"] = planning.TaskResult{ElapsedMinutes: 600}
			return s
		}(),
	}

	svc := application.NewBillingService(repo)

	status, err := svc.GetBudgetStatus()
	if err != nil {
		t.Fatalf("GetBudgetStatus failed: %v", err)
	}
	if status == nil {
		t.Fatal("expected status, got nil")
	}
	if status.UsedHours != 20.0 {
		t.Errorf("expected 20.0 used hours, got %f", status.UsedHours)
	}
	if status.OverBudget {
		t.Error("should not be over budget")
	}
}

func TestBillingService_GetBudgetStatus_NoBudget(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		Policy:   &domain.PolicyConfig{BudgetHours: 0},
	}

	svc := application.NewBillingService(repo)

	status, err := svc.GetBudgetStatus()
	if err != nil {
		t.Fatalf("GetBudgetStatus failed: %v", err)
	}
	if status != nil {
		t.Error("expected nil when no budget set")
	}
}

func TestBillingService_LogTime(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100, IsDefault: true},
			},
		},
	}

	svc := application.NewBillingService(repo)

	err := svc.LogTime("task-1", "rate-1", 60, "Worked on task")
	if err != nil {
		t.Fatalf("LogTime failed: %v", err)
	}

	entries, _ := repo.LoadTimeEntries()
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestBillingService_LogTime_NoRate(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo:    &MockRepo{},
		RatesConfig: &billing.RateConfig{},
	}

	svc := application.NewBillingService(repo)

	err := svc.LogTime("task-1", "", 60, "Worked on task")
	if err == nil {
		t.Fatal("expected error when no rate specified and no default")
	}
}

func TestBillingService_LogTime_NegativeMinutes(t *testing.T) {
	repo := &mockBillingRepo{MockRepo: &MockRepo{}}

	svc := application.NewBillingService(repo)

	err := svc.LogTime("task-1", "rate-1", -10, "Worked on task")
	if err == nil {
		t.Fatal("expected error for negative minutes")
	}
}

func TestBillingService_StartTask(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100, IsDefault: true},
			},
		},
		ExecState: planning.NewExecutionState("test"),
	}

	svc := application.NewBillingService(repo)

	err := svc.StartTask("task-1", "rate-1")
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	state, _ := repo.LoadState()
	_, ok := state.TaskStates["task-1"]
	if !ok {
		t.Error("expected task-1 in state")
	}
}

func TestBillingService_StartTask_AlreadyInProgress(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		ExecState: func() *planning.ExecutionState {
			s := planning.NewExecutionState("test")
			s.SetTaskStatus("task-1", planning.StatusInProgress)
			return s
		}(),
	}

	svc := application.NewBillingService(repo)

	err := svc.StartTask("task-1", "")
	if err == nil {
		t.Fatal("expected error for already in progress task")
	}
}

func TestBillingService_CompleteTask(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100, IsDefault: true},
			},
		},
		ExecState: func() *planning.ExecutionState {
			s := planning.NewExecutionState("test")
			s.SetTaskStatus("task-1", planning.StatusInProgress)
			taskResult, _ := s.GetTaskResult("task-1")
			taskResult.ElapsedMinutes = 60
			s.TaskStates["task-1"] = taskResult
			return s
		}(),
	}

	svc := application.NewBillingService(repo)

	err := svc.CompleteTask("task-1")
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	entries, _ := repo.LoadTimeEntries()
	if len(entries) != 1 {
		t.Errorf("expected 1 time entry, got %d", len(entries))
	}
}

func TestBillingService_CompleteTask_NotInProgress(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo:  &MockRepo{},
		ExecState: planning.NewExecutionState("test"),
	}

	svc := application.NewBillingService(repo)

	err := svc.CompleteTask("task-1")
	if err == nil {
		t.Fatal("expected error for task not in progress")
	}
}

func TestBillingService_ListRates(t *testing.T) {
	repo := &mockBillingRepo{
		MockRepo: &MockRepo{},
		RatesConfig: &billing.RateConfig{
			Rates: []billing.Rate{
				{ID: "rate-1", Name: "Standard", HourlyRate: 100},
			},
		},
	}

	svc := application.NewBillingService(repo)

	config, err := svc.ListRates()
	if err != nil {
		t.Fatalf("ListRates failed: %v", err)
	}
	if len(config.Rates) != 1 {
		t.Errorf("expected 1 rate, got %d", len(config.Rates))
	}
}
