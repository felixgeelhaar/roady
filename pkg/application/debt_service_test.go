package application

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/debt"
	"github.com/felixgeelhaar/roady/pkg/domain/drift"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func setupDebtTestServices(t *testing.T) (*DebtService, *DriftService) {
	t.Helper()

	tmpDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	// Set up a basic spec with features
	testSpec := &spec.ProductSpec{
		ID:    "test-spec",
		Title: "Test Project",
		Features: []spec.Feature{
			{ID: "feature-1", Title: "Feature 1"},
			{ID: "feature-2", Title: "Feature 2"},
		},
	}
	if err := repo.SaveSpec(testSpec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	// Create services
	auditSvc := NewAuditService(repo)
	policySvc := NewPolicyService(repo)
	inspector := storage.NewCodebaseInspector()
	driftSvc := NewDriftService(repo, auditSvc, inspector, policySvc)

	debtSvc := NewDebtService(driftSvc, auditSvc)

	return debtSvc, driftSvc
}

func TestNewDebtService(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)

	if debtSvc == nil {
		t.Fatal("Expected non-nil debt service")
	}
	if debtSvc.driftHistoryProj == nil {
		t.Fatal("Expected non-nil drift history projection")
	}
}

func TestDebtService_GetDebtReport(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)
	ctx := context.Background()

	report, err := debtSvc.GetDebtReport(ctx)
	if err != nil {
		t.Fatalf("GetDebtReport failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected non-nil report")
	}
	// With no plan, we should have drift for missing tasks
	// The exact number depends on the drift detection rules
}

func TestDebtService_GetStickyDrift(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)

	items, err := debtSvc.GetStickyDrift()
	if err != nil {
		t.Fatalf("GetStickyDrift failed: %v", err)
	}

	// Initially no sticky drift (nothing old enough)
	if len(items) != 0 {
		t.Errorf("Expected 0 sticky items initially, got %d", len(items))
	}
}

func TestDebtService_GetDebtByComponent(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)

	byComponent, err := debtSvc.GetDebtByComponent()
	if err != nil {
		t.Fatalf("GetDebtByComponent failed: %v", err)
	}

	if byComponent == nil {
		t.Fatal("Expected non-nil map")
	}
}

func TestDebtService_GetDebtByCategory(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)

	byCategory, err := debtSvc.GetDebtByCategory()
	if err != nil {
		t.Fatalf("GetDebtByCategory failed: %v", err)
	}

	if byCategory == nil {
		t.Fatal("Expected non-nil map")
	}
}

func TestDebtService_GetDebtScore(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)

	score, err := debtSvc.GetDebtScore("feature-1")
	if err != nil {
		t.Fatalf("GetDebtScore failed: %v", err)
	}

	if score == nil {
		t.Fatal("Expected non-nil score")
	}
	if score.ComponentID != "feature-1" {
		t.Errorf("ComponentID = %s, want feature-1", score.ComponentID)
	}
}

func TestDebtService_GetDriftHistory(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)

	// Test with no window (all history)
	history, err := debtSvc.GetDriftHistory(0)
	if err != nil {
		t.Fatalf("GetDriftHistory failed: %v", err)
	}

	if history == nil {
		t.Fatal("Expected non-nil history")
	}

	// Test with window
	historyWindow, err := debtSvc.GetDriftHistory(7)
	if err != nil {
		t.Fatalf("GetDriftHistory with window failed: %v", err)
	}

	if historyWindow == nil {
		t.Fatal("Expected non-nil history with window")
	}
}

func TestDebtService_GetDriftTrend(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)

	// Test with default window
	trend, err := debtSvc.GetDriftTrend(0)
	if err != nil {
		t.Fatalf("GetDriftTrend failed: %v", err)
	}

	if trend.WindowDays != 30 {
		t.Errorf("WindowDays = %d, want 30 (default)", trend.WindowDays)
	}

	// Test with specific window
	trend14, err := debtSvc.GetDriftTrend(14)
	if err != nil {
		t.Fatalf("GetDriftTrend with 14 days failed: %v", err)
	}

	if trend14.WindowDays != 14 {
		t.Errorf("WindowDays = %d, want 14", trend14.WindowDays)
	}
}

func TestDebtService_GetTopDebtors(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)
	ctx := context.Background()

	top, err := debtSvc.GetTopDebtors(ctx, 5)
	if err != nil {
		t.Fatalf("GetTopDebtors failed: %v", err)
	}

	// May have debtors depending on drift, nil is valid when no debt exists
	// Just ensure no error occurred
	_ = top
}

func TestDebtService_GetHealthLevel(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)
	ctx := context.Background()

	level, err := debtSvc.GetHealthLevel(ctx)
	if err != nil {
		t.Fatalf("GetHealthLevel failed: %v", err)
	}

	validLevels := map[string]bool{
		"healthy":    true,
		"moderate":   true,
		"concerning": true,
		"critical":   true,
	}

	if !validLevels[level] {
		t.Errorf("Invalid health level: %s", level)
	}
}

func TestDebtService_RecordDriftDetection(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)
	ctx := context.Background()

	driftReport := &drift.Report{
		ID:        "test-report",
		CreatedAt: time.Now(),
		Issues: []drift.Issue{
			{
				ID:          "issue-1",
				Type:        drift.DriftTypePlan,
				Category:    drift.CategoryMissing,
				ComponentID: "feature-1",
				Message:     "Test drift",
			},
		},
	}

	err := debtSvc.RecordDriftDetection(ctx, driftReport)
	if err != nil {
		t.Fatalf("RecordDriftDetection failed: %v", err)
	}

	// Verify it was recorded
	history, _ := debtSvc.GetDriftHistory(0)
	if len(history) != 1 {
		t.Errorf("len(history) = %d, want 1", len(history))
	}
}

func TestDebtService_RecordDriftAccepted(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)
	ctx := context.Background()

	// First record a drift
	driftReport := &drift.Report{
		ID:        "test-report",
		CreatedAt: time.Now(),
		Issues: []drift.Issue{
			{
				ID:          "issue-1",
				Type:        drift.DriftTypePlan,
				Category:    drift.CategoryMissing,
				ComponentID: "feature-1",
				Message:     "Test drift",
			},
		},
	}
	debtSvc.RecordDriftDetection(ctx, driftReport)

	// Accept the drift
	err := debtSvc.RecordDriftAccepted("feature-1", drift.DriftTypePlan)
	if err != nil {
		t.Fatalf("RecordDriftAccepted failed: %v", err)
	}

	// Check the category changed
	byCategory, _ := debtSvc.GetDebtByCategory()
	if len(byCategory[debt.DebtIntentional]) != 1 {
		t.Errorf("Expected 1 intentional debt item")
	}
}

func TestDebtService_RecordDriftResolved(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)
	ctx := context.Background()

	// First record a drift
	driftReport := &drift.Report{
		ID:        "test-report",
		CreatedAt: time.Now(),
		Issues: []drift.Issue{
			{
				ID:          "issue-1",
				Type:        drift.DriftTypePlan,
				Category:    drift.CategoryMissing,
				ComponentID: "feature-1",
				Message:     "Test drift",
			},
		},
	}
	debtSvc.RecordDriftDetection(ctx, driftReport)

	// Verify it's active
	byComponent, _ := debtSvc.GetDebtByComponent()
	if len(byComponent["feature-1"]) != 1 {
		t.Fatal("Expected 1 active debt item")
	}

	// Resolve the drift
	err := debtSvc.RecordDriftResolved("feature-1", drift.DriftTypePlan)
	if err != nil {
		t.Fatalf("RecordDriftResolved failed: %v", err)
	}

	// Verify it's resolved (no longer active)
	byComponent, _ = debtSvc.GetDebtByComponent()
	if len(byComponent["feature-1"]) != 0 {
		t.Errorf("Expected 0 active debt items after resolution")
	}
}

func TestDebtService_GetDebtSummary(t *testing.T) {
	debtSvc, _ := setupDebtTestServices(t)
	ctx := context.Background()

	summary, err := debtSvc.GetDebtSummary(ctx)
	if err != nil {
		t.Fatalf("GetDebtSummary failed: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected non-nil summary")
	}

	validLevels := map[string]bool{
		"healthy":    true,
		"moderate":   true,
		"concerning": true,
		"critical":   true,
	}
	if !validLevels[summary.HealthLevel] {
		t.Errorf("Invalid health level in summary: %s", summary.HealthLevel)
	}
}

func TestMapDriftCategoryToDebtCategory(t *testing.T) {
	tests := []struct {
		driftCategory drift.DriftCategory
		expected      debt.DebtCategory
	}{
		{drift.CategoryDebt, debt.DebtIntentional},
		{drift.CategoryMissing, debt.DebtNeglect},
		{drift.CategoryOrphan, debt.DebtChurn},
		{drift.CategoryMismatch, debt.DebtNeglect},
		{drift.CategoryViolation, debt.DebtNeglect},
		{drift.CategoryImplementation, debt.DebtChurn},
		{drift.DriftCategory("unknown"), debt.DebtNeglect},
	}

	for _, tt := range tests {
		t.Run(string(tt.driftCategory), func(t *testing.T) {
			result := mapDriftCategoryToDebtCategory(tt.driftCategory)
			if result != tt.expected {
				t.Errorf("mapDriftCategoryToDebtCategory(%s) = %s, want %s",
					tt.driftCategory, result, tt.expected)
			}
		})
	}
}
