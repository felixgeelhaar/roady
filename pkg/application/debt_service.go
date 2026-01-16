package application

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/domain/debt"
	"github.com/felixgeelhaar/roady/pkg/domain/drift"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
)

// DebtService provides debt analysis capabilities.
type DebtService struct {
	driftSvc          *DriftService
	driftHistoryProj  *events.DriftHistoryProjection
	auditSvc          *AuditService
}

// NewDebtService creates a new debt service.
func NewDebtService(driftSvc *DriftService, auditSvc *AuditService) *DebtService {
	return &DebtService{
		driftSvc:         driftSvc,
		driftHistoryProj: events.NewDriftHistoryProjection(),
		auditSvc:         auditSvc,
	}
}

// GetDebtReport generates a comprehensive debt report based on current drift.
func (s *DebtService) GetDebtReport(ctx context.Context) (*debt.DebtReport, error) {
	// Get current drift
	driftReport, err := s.driftSvc.DetectDrift(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect drift: %w", err)
	}

	// Build debt report from drift issues
	report := debt.NewDebtReport()

	for _, issue := range driftReport.Issues {
		item := debt.NewDebtItem(issue.ComponentID, issue.Type, issue.Message)
		item.SetCategory(mapDriftCategoryToDebtCategory(issue.Category))
		report.AddItem(item)
	}

	report.Finalize()
	return report, nil
}

// mapDriftCategoryToDebtCategory converts drift categories to debt categories.
func mapDriftCategoryToDebtCategory(category drift.DriftCategory) debt.DebtCategory {
	switch category {
	case drift.CategoryDebt:
		return debt.DebtIntentional
	case drift.CategoryMissing:
		return debt.DebtNeglect
	case drift.CategoryOrphan:
		return debt.DebtChurn
	case drift.CategoryMismatch:
		return debt.DebtNeglect
	case drift.CategoryViolation:
		return debt.DebtNeglect
	case drift.CategoryImplementation:
		return debt.DebtChurn
	default:
		return debt.DebtNeglect
	}
}

// GetStickyDrift returns all sticky debt items (unresolved >7 days).
func (s *DebtService) GetStickyDrift() ([]*debt.DebtItem, error) {
	return s.driftHistoryProj.GetStickyDebtItems(), nil
}

// GetDebtByComponent returns debt items grouped by component.
func (s *DebtService) GetDebtByComponent() (map[string][]*debt.DebtItem, error) {
	return s.driftHistoryProj.GetDebtByComponent(), nil
}

// GetDebtByCategory returns debt items grouped by category.
func (s *DebtService) GetDebtByCategory() (map[debt.DebtCategory][]*debt.DebtItem, error) {
	return s.driftHistoryProj.GetDebtByCategory(), nil
}

// GetDebtScore calculates the debt score for a specific component.
func (s *DebtService) GetDebtScore(componentID string) (*debt.DebtScore, error) {
	byComponent := s.driftHistoryProj.GetDebtByComponent()
	items, ok := byComponent[componentID]
	if !ok {
		return debt.NewDebtScore(componentID, nil), nil
	}
	return debt.NewDebtScore(componentID, items), nil
}

// GetDriftHistory returns historical drift snapshots.
func (s *DebtService) GetDriftHistory(windowDays int) ([]events.DriftSnapshot, error) {
	if windowDays <= 0 {
		return s.driftHistoryProj.GetDriftHistory(), nil
	}
	return s.driftHistoryProj.GetDriftHistoryInWindow(windowDays), nil
}

// GetDriftTrend analyzes drift patterns over time.
func (s *DebtService) GetDriftTrend(windowDays int) (events.DriftTrend, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	return s.driftHistoryProj.GetDriftTrend(windowDays), nil
}

// GetTopDebtors returns the components with the highest debt scores.
func (s *DebtService) GetTopDebtors(ctx context.Context, limit int) ([]*debt.DebtScore, error) {
	report, err := s.GetDebtReport(ctx)
	if err != nil {
		return nil, err
	}
	return report.GetTopDebtors(limit), nil
}

// GetHealthLevel returns an overall health assessment based on debt.
func (s *DebtService) GetHealthLevel(ctx context.Context) (string, error) {
	report, err := s.GetDebtReport(ctx)
	if err != nil {
		return "", err
	}
	return report.GetHealthLevel(), nil
}

// RecordDriftDetection records a drift detection event for historical tracking.
func (s *DebtService) RecordDriftDetection(ctx context.Context, driftReport *drift.Report) error {
	for _, issue := range driftReport.Issues {
		event := &events.BaseEvent{
			ID:        driftReport.ID + "-" + issue.ID,
			Type:      events.EventTypeDriftDetected,
			Timestamp: driftReport.CreatedAt,
			Metadata: map[string]any{
				"component_id": issue.ComponentID,
				"drift_type":   string(issue.Type),
				"category":     string(issue.Category),
				"message":      issue.Message,
				"issue_count":  len(driftReport.Issues),
			},
		}
		if err := s.driftHistoryProj.Apply(event); err != nil {
			return fmt.Errorf("apply drift event: %w", err)
		}
	}
	return nil
}

// RecordDriftAccepted records a drift acceptance event.
func (s *DebtService) RecordDriftAccepted(componentID string, driftType drift.DriftType) error {
	event := &events.BaseEvent{
		Type: events.EventTypeDriftAccepted,
		Metadata: map[string]any{
			"component_id": componentID,
			"drift_type":   string(driftType),
		},
	}
	return s.driftHistoryProj.Apply(event)
}

// RecordDriftResolved records a drift resolution event.
func (s *DebtService) RecordDriftResolved(componentID string, driftType drift.DriftType) error {
	event := &events.BaseEvent{
		Type: events.EventTypeDriftResolved,
		Metadata: map[string]any{
			"component_id": componentID,
			"drift_type":   string(driftType),
		},
	}
	return s.driftHistoryProj.Apply(event)
}

// DebtSummary provides a quick overview of debt status.
type DebtSummary struct {
	TotalItems      int     `json:"total_items"`
	StickyItems     int     `json:"sticky_items"`
	AverageScore    float64 `json:"average_score"`
	HealthLevel     string  `json:"health_level"`
	TopDebtor       string  `json:"top_debtor,omitempty"`
	TopDebtorScore  float64 `json:"top_debtor_score,omitempty"`
}

// GetDebtSummary returns a quick overview of the debt status.
func (s *DebtService) GetDebtSummary(ctx context.Context) (*DebtSummary, error) {
	report, err := s.GetDebtReport(ctx)
	if err != nil {
		return nil, err
	}

	summary := &DebtSummary{
		TotalItems:   report.TotalItems,
		StickyItems:  report.StickyItems,
		AverageScore: report.AverageScore,
		HealthLevel:  report.GetHealthLevel(),
	}

	topDebtors := report.GetTopDebtors(1)
	if len(topDebtors) > 0 {
		summary.TopDebtor = topDebtors[0].ComponentID
		summary.TopDebtorScore = topDebtors[0].Score
	}

	return summary, nil
}
