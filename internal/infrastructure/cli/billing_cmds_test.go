package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/domain/team"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// ---------------------------------------------------------------------------
// Team command tests
// ---------------------------------------------------------------------------

func TestTeamListCmd_Empty(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	output := captureStdout(t, func() {
		if err := teamListCmd.RunE(teamListCmd, []string{}); err != nil {
			t.Fatalf("team list failed: %v", err)
		}
	})

	if !strings.Contains(output, "No team members configured") {
		t.Fatalf("expected empty team output, got:\n%s", output)
	}
}

func TestTeamListCmd_WithMembers(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveTeam(&team.TeamConfig{
		Members: []team.Member{
			{Name: "alice", Role: team.RoleAdmin},
			{Name: "bob", Role: team.RoleMember},
		},
	})

	output := captureStdout(t, func() {
		if err := teamListCmd.RunE(teamListCmd, []string{}); err != nil {
			t.Fatalf("team list failed: %v", err)
		}
	})

	if !strings.Contains(output, "alice") {
		t.Fatalf("expected alice in output, got:\n%s", output)
	}
	if !strings.Contains(output, "bob") {
		t.Fatalf("expected bob in output, got:\n%s", output)
	}
}

func TestTeamAddCmd(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	_ = repo.SaveSpec(&spec.ProductSpec{
		ID:    "spec-1",
		Title: "Project",
	})

	output := captureStdout(t, func() {
		if err := teamAddCmd.RunE(teamAddCmd, []string{"alice", "admin"}); err != nil {
			t.Fatalf("team add failed: %v", err)
		}
	})

	if !strings.Contains(output, "alice") || !strings.Contains(output, "added") {
		t.Fatalf("expected member added output, got:\n%s", output)
	}
}

func TestTeamRemoveCmd(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveTeam(&team.TeamConfig{
		Members: []team.Member{
			{Name: "alice", Role: team.RoleAdmin},
		},
	})

	output := captureStdout(t, func() {
		if err := teamRemoveCmd.RunE(teamRemoveCmd, []string{"alice"}); err != nil {
			t.Fatalf("team remove failed: %v", err)
		}
	})

	if !strings.Contains(output, "alice") || !strings.Contains(output, "removed") {
		t.Fatalf("expected member removed output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// Cost command tests
// ---------------------------------------------------------------------------

func TestCostReportCmd_Empty(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	_ = repo.SavePlan(&planning.Plan{ID: "p1"})
	_ = repo.SaveState(planning.NewExecutionState("p1"))

	// Reset flag vars to defaults.
	costFormat = "text"
	costTaskID = ""
	costPeriod = ""
	costOutput = ""
	defer func() {
		costFormat = "text"
		costTaskID = ""
		costPeriod = ""
		costOutput = ""
	}()

	output := captureStdout(t, func() {
		if err := costReportCmd.RunE(costReportCmd, []string{}); err != nil {
			t.Fatalf("cost report failed: %v", err)
		}
	})

	if !strings.Contains(output, "No time entries found") {
		t.Fatalf("expected empty report output, got:\n%s", output)
	}
}

func TestCostReportCmd_WithEntries(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveRates(&billing.RateConfig{
		Currency: "USD",
		Rates: []billing.Rate{
			{ID: "std", Name: "Standard", HourlyRate: 100, IsDefault: true},
		},
	})
	_ = repo.SaveTimeEntries([]billing.TimeEntry{
		{ID: "te-1", TaskID: "t1", RateID: "std", Minutes: 150, Description: "Work", CreatedAt: time.Now()},
	})
	_ = repo.SavePlan(&planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", FeatureID: "f1", Title: "Task One"},
		},
	})
	_ = repo.SaveState(planning.NewExecutionState("p1"))

	// Reset flag vars.
	costFormat = "text"
	costTaskID = ""
	costPeriod = ""
	costOutput = ""
	defer func() {
		costFormat = "text"
		costTaskID = ""
		costPeriod = ""
		costOutput = ""
	}()

	output := captureStdout(t, func() {
		if err := costReportCmd.RunE(costReportCmd, []string{}); err != nil {
			t.Fatalf("cost report failed: %v", err)
		}
	})

	if !strings.Contains(output, "Cost Report") {
		t.Fatalf("expected cost report output, got:\n%s", output)
	}
}

func TestCostBudgetCmd_NoBudget(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	_ = repo.SavePlan(&planning.Plan{ID: "p1"})
	_ = repo.SaveState(planning.NewExecutionState("p1"))

	output := captureStdout(t, func() {
		if err := costBudgetCmd.RunE(costBudgetCmd, []string{}); err != nil {
			t.Fatalf("cost budget failed: %v", err)
		}
	})

	if !strings.Contains(output, "No budget configured") {
		t.Fatalf("expected no budget output, got:\n%s", output)
	}
}

func TestCostBudgetCmd_WithBudget(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SavePolicy(&domain.PolicyConfig{BudgetHours: 40})
	_ = repo.SaveRates(&billing.RateConfig{
		Currency: "USD",
		Rates: []billing.Rate{
			{ID: "std", Name: "Standard", HourlyRate: 100, IsDefault: true},
		},
	})
	_ = repo.SaveTimeEntries([]billing.TimeEntry{
		{ID: "te-1", TaskID: "t1", RateID: "std", Minutes: 120, Description: "Work", CreatedAt: time.Now()},
	})
	_ = repo.SavePlan(&planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", FeatureID: "f1", Title: "Task One"},
		},
	})
	_ = repo.SaveState(planning.NewExecutionState("p1"))

	output := captureStdout(t, func() {
		if err := costBudgetCmd.RunE(costBudgetCmd, []string{}); err != nil {
			t.Fatalf("cost budget failed: %v", err)
		}
	})

	if !strings.Contains(output, "Budget Status") {
		t.Fatalf("expected budget status output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// Rate command tests
// ---------------------------------------------------------------------------

func TestRateListCmd_Empty(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	output := captureStdout(t, func() {
		if err := rateListCmd.RunE(rateListCmd, []string{}); err != nil {
			t.Fatalf("rate list failed: %v", err)
		}
	})

	if !strings.Contains(output, "No rates configured") {
		t.Fatalf("expected no rates output, got:\n%s", output)
	}
}

func TestRateAddCmd(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	// Set flag vars for the add command.
	rateID = "senior"
	rateName = "Senior Dev"
	rateAmount = 150.0
	rateDefault = true
	defer func() {
		rateID = ""
		rateName = ""
		rateAmount = 0
		rateDefault = false
	}()

	output := captureStdout(t, func() {
		if err := rateAddCmd.RunE(rateAddCmd, []string{}); err != nil {
			t.Fatalf("rate add failed: %v", err)
		}
	})

	if !strings.Contains(output, "Added rate") {
		t.Fatalf("expected added rate output, got:\n%s", output)
	}
}

func TestRateListCmd_WithRates(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveRates(&billing.RateConfig{
		Currency: "USD",
		Rates: []billing.Rate{
			{ID: "senior", Name: "Senior Dev", HourlyRate: 150, IsDefault: true},
			{ID: "junior", Name: "Junior Dev", HourlyRate: 80, IsDefault: false},
		},
	})

	output := captureStdout(t, func() {
		if err := rateListCmd.RunE(rateListCmd, []string{}); err != nil {
			t.Fatalf("rate list failed: %v", err)
		}
	})

	if !strings.Contains(output, "senior") || !strings.Contains(output, "Senior Dev") {
		t.Fatalf("expected rate info in output, got:\n%s", output)
	}
	if !strings.Contains(output, "(default)") {
		t.Fatalf("expected default marker in output, got:\n%s", output)
	}
}

func TestRateRemoveCmd(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveRates(&billing.RateConfig{
		Currency: "USD",
		Rates: []billing.Rate{
			{ID: "senior", Name: "Senior Dev", HourlyRate: 150, IsDefault: true},
		},
	})

	output := captureStdout(t, func() {
		if err := rateRemoveCmd.RunE(rateRemoveCmd, []string{"senior"}); err != nil {
			t.Fatalf("rate remove failed: %v", err)
		}
	})

	if !strings.Contains(output, "Removed rate") {
		t.Fatalf("expected removed rate output, got:\n%s", output)
	}
}

func TestRateSetDefaultCmd(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveRates(&billing.RateConfig{
		Currency: "USD",
		Rates: []billing.Rate{
			{ID: "senior", Name: "Senior Dev", HourlyRate: 150, IsDefault: false},
			{ID: "junior", Name: "Junior Dev", HourlyRate: 80, IsDefault: false},
		},
	})

	output := captureStdout(t, func() {
		if err := rateSetDefaultCmd.RunE(rateSetDefaultCmd, []string{"senior"}); err != nil {
			t.Fatalf("rate set default failed: %v", err)
		}
	})

	if !strings.Contains(output, "Set default rate") {
		t.Fatalf("expected set default output, got:\n%s", output)
	}
}

func TestRateTaxSetCmd(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveRates(&billing.RateConfig{
		Currency: "USD",
		Rates: []billing.Rate{
			{ID: "std", Name: "Standard", HourlyRate: 100, IsDefault: true},
		},
	})

	// Set flag vars for the tax set command.
	taxName = "VAT"
	taxPercent = 20.0
	taxIncluded = false
	defer func() {
		taxName = ""
		taxPercent = 0
		taxIncluded = false
	}()

	output := captureStdout(t, func() {
		if err := rateTaxSetCmd.RunE(rateTaxSetCmd, []string{}); err != nil {
			t.Fatalf("rate tax set failed: %v", err)
		}
	})

	if !strings.Contains(output, "Tax configured") {
		t.Fatalf("expected tax configured output, got:\n%s", output)
	}
}
