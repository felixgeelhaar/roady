package debt

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/drift"
)

func TestDebtCategory_IsValid(t *testing.T) {
	tests := []struct {
		category DebtCategory
		want     bool
	}{
		{DebtIntentional, true},
		{DebtRegression, true},
		{DebtNeglect, true},
		{DebtChurn, true},
		{DebtCategory("invalid"), false},
		{DebtCategory(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			if got := tt.category.IsValid(); got != tt.want {
				t.Errorf("DebtCategory(%q).IsValid() = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}

func TestNewDebtItem(t *testing.T) {
	item := NewDebtItem("feature-1", drift.DriftTypePlan, "Missing task for feature")

	if item.ComponentID != "feature-1" {
		t.Errorf("ComponentID = %s, want feature-1", item.ComponentID)
	}
	if item.DriftType != drift.DriftTypePlan {
		t.Errorf("DriftType = %s, want %s", item.DriftType, drift.DriftTypePlan)
	}
	if item.DetectionCount != 1 {
		t.Errorf("DetectionCount = %d, want 1", item.DetectionCount)
	}
	if item.Category != DebtNeglect {
		t.Errorf("Category = %s, want %s", item.Category, DebtNeglect)
	}
	if item.IsSticky {
		t.Error("New item should not be sticky")
	}
}

func TestDebtItem_Update(t *testing.T) {
	item := NewDebtItem("feature-1", drift.DriftTypePlan, "Missing task")
	item.FirstDetected = time.Now().Add(-10 * 24 * time.Hour) // 10 days ago

	item.Update()

	if item.DetectionCount != 2 {
		t.Errorf("DetectionCount = %d, want 2", item.DetectionCount)
	}
	if !item.IsSticky {
		t.Error("Item older than 7 days should be sticky")
	}
	if item.DaysPending < 9 {
		t.Errorf("DaysPending = %d, want >= 9", item.DaysPending)
	}
}

func TestDebtItem_Resolve(t *testing.T) {
	item := NewDebtItem("feature-1", drift.DriftTypePlan, "Missing task")

	if item.IsResolved() {
		t.Error("New item should not be resolved")
	}

	item.Resolve()

	if !item.IsResolved() {
		t.Error("Item should be resolved after Resolve()")
	}
	if item.ResolvedAt == nil {
		t.Error("ResolvedAt should be set")
	}
}

func TestDebtItem_SetCategory(t *testing.T) {
	item := NewDebtItem("feature-1", drift.DriftTypePlan, "Missing task")

	item.SetCategory(DebtIntentional)

	if item.Category != DebtIntentional {
		t.Errorf("Category = %s, want %s", item.Category, DebtIntentional)
	}
}

func TestNewDebtScore(t *testing.T) {
	items := []*DebtItem{
		{
			ComponentID:    "feature-1",
			DriftType:      drift.DriftTypePlan,
			DaysPending:    5,
			DetectionCount: 2,
			IsSticky:       false,
		},
		{
			ComponentID:    "feature-1",
			DriftType:      drift.DriftTypeSpec,
			DaysPending:    10,
			DetectionCount: 3,
			IsSticky:       true,
		},
	}

	score := NewDebtScore("feature-1", items)

	if score.ComponentID != "feature-1" {
		t.Errorf("ComponentID = %s, want feature-1", score.ComponentID)
	}
	if len(score.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(score.Items))
	}
	if score.StickyCount != 1 {
		t.Errorf("StickyCount = %d, want 1", score.StickyCount)
	}
	if score.TotalDaysPending != 15 {
		t.Errorf("TotalDaysPending = %d, want 15", score.TotalDaysPending)
	}
	if score.Score <= 0 {
		t.Errorf("Score should be > 0, got %f", score.Score)
	}
}

func TestDebtScore_Calculate_Empty(t *testing.T) {
	score := &DebtScore{
		ComponentID: "empty",
		Items:       []*DebtItem{},
	}

	score.Calculate()

	if score.Score != 0 {
		t.Errorf("Score = %f, want 0 for empty items", score.Score)
	}
	if score.StickyCount != 0 {
		t.Errorf("StickyCount = %d, want 0", score.StickyCount)
	}
}

func TestDebtScore_Calculate_Cap(t *testing.T) {
	// Create many high-penalty items
	items := make([]*DebtItem, 20)
	for i := range items {
		items[i] = &DebtItem{
			DaysPending:    100,
			DetectionCount: 10,
			IsSticky:       true,
		}
	}

	score := NewDebtScore("high-debt", items)

	if score.Score > 100 {
		t.Errorf("Score = %f, should be capped at 100", score.Score)
	}
}

func TestNewDebtReport(t *testing.T) {
	report := NewDebtReport()

	if report.Scores == nil {
		t.Error("Scores should be initialized")
	}
	if report.ByCategory == nil {
		t.Error("ByCategory should be initialized")
	}
	if report.ByDriftType == nil {
		t.Error("ByDriftType should be initialized")
	}
	if report.StickyDrift == nil {
		t.Error("StickyDrift should be initialized")
	}
}

func TestDebtReport_AddItem(t *testing.T) {
	report := NewDebtReport()

	item := &DebtItem{
		ComponentID: "feature-1",
		DriftType:   drift.DriftTypePlan,
		Category:    DebtNeglect,
		IsSticky:    true,
	}

	report.AddItem(item)

	if report.TotalItems != 1 {
		t.Errorf("TotalItems = %d, want 1", report.TotalItems)
	}
	if report.StickyItems != 1 {
		t.Errorf("StickyItems = %d, want 1", report.StickyItems)
	}
	if len(report.StickyDrift) != 1 {
		t.Errorf("len(StickyDrift) = %d, want 1", len(report.StickyDrift))
	}
	if len(report.ByCategory[DebtNeglect]) != 1 {
		t.Errorf("ByCategory[neglect] = %d, want 1", len(report.ByCategory[DebtNeglect]))
	}
	if len(report.ByDriftType[drift.DriftTypePlan]) != 1 {
		t.Errorf("ByDriftType[missing_task] = %d, want 1", len(report.ByDriftType[drift.DriftTypePlan]))
	}
	if _, ok := report.Scores["feature-1"]; !ok {
		t.Error("Scores should contain feature-1")
	}
}

func TestDebtReport_Finalize(t *testing.T) {
	report := NewDebtReport()

	// Add items for two components
	report.AddItem(&DebtItem{
		ComponentID:    "feature-1",
		DriftType:      drift.DriftTypePlan,
		Category:       DebtNeglect,
		DaysPending:    5,
		DetectionCount: 1,
	})
	report.AddItem(&DebtItem{
		ComponentID:    "feature-2",
		DriftType:      drift.DriftTypeSpec,
		Category:       DebtChurn,
		DaysPending:    20,
		DetectionCount: 5,
		IsSticky:       true,
	})

	report.Finalize()

	if report.AverageScore <= 0 {
		t.Errorf("AverageScore should be > 0, got %f", report.AverageScore)
	}
	if report.MaxScore <= 0 {
		t.Errorf("MaxScore should be > 0, got %f", report.MaxScore)
	}
	if report.Scores["feature-1"].Score == 0 {
		t.Error("feature-1 should have a non-zero score")
	}
	if report.Scores["feature-2"].Score == 0 {
		t.Error("feature-2 should have a non-zero score")
	}
}

func TestDebtReport_GetHealthLevel(t *testing.T) {
	tests := []struct {
		avgScore float64
		want     string
	}{
		{10, "healthy"},
		{35, "moderate"},
		{65, "concerning"},
		{90, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			report := &DebtReport{AverageScore: tt.avgScore}
			if got := report.GetHealthLevel(); got != tt.want {
				t.Errorf("GetHealthLevel() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDebtReport_GetTopDebtors(t *testing.T) {
	report := NewDebtReport()

	// Add items for three components with different scores
	for i := 0; i < 3; i++ {
		for j := 0; j <= i; j++ {
			report.AddItem(&DebtItem{
				ComponentID:    string(rune('a' + i)), // a, b, c
				DriftType:      drift.DriftTypePlan,
				DaysPending:    10 * (i + 1),
				DetectionCount: i + 1,
				IsSticky:       i > 0,
			})
		}
	}
	report.Finalize()

	top := report.GetTopDebtors(2)

	if len(top) != 2 {
		t.Fatalf("len(top) = %d, want 2", len(top))
	}
	// c should have highest score (most items, most days, sticky)
	if top[0].ComponentID != "c" {
		t.Errorf("Top debtor = %s, want c", top[0].ComponentID)
	}
}

func TestDebtReport_GetTopDebtors_Empty(t *testing.T) {
	report := NewDebtReport()

	top := report.GetTopDebtors(5)

	if top != nil {
		t.Errorf("Expected nil for empty report, got %v", top)
	}
}

func TestDebtReport_GetTopDebtors_LessThanLimit(t *testing.T) {
	report := NewDebtReport()
	report.AddItem(&DebtItem{
		ComponentID: "only-one",
		DriftType:   drift.DriftTypePlan,
	})
	report.Finalize()

	top := report.GetTopDebtors(5)

	if len(top) != 1 {
		t.Errorf("len(top) = %d, want 1", len(top))
	}
}

func TestDebtReport_Finalize_Empty(t *testing.T) {
	report := NewDebtReport()
	report.Finalize()

	if report.AverageScore != 0 {
		t.Errorf("AverageScore = %f, want 0", report.AverageScore)
	}
	if report.MaxScore != 0 {
		t.Errorf("MaxScore = %f, want 0", report.MaxScore)
	}
}
