// Package debt provides types for tracking and analyzing planning debt.
// Planning debt represents recurring drift patterns and unresolved issues
// that accumulate over time.
package debt

import (
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/drift"
)

// DebtCategory classifies the origin of planning debt.
type DebtCategory string

const (
	// DebtIntentional represents deliberate technical decisions with known trade-offs.
	DebtIntentional DebtCategory = "intentional"
	// DebtRegression represents previously resolved issues that have returned.
	DebtRegression DebtCategory = "regression"
	// DebtNeglect represents issues left unaddressed over time.
	DebtNeglect DebtCategory = "neglect"
	// DebtChurn represents repeatedly changing requirements or implementation.
	DebtChurn DebtCategory = "churn"
)

// IsValid returns true if the category is a recognized debt category.
func (c DebtCategory) IsValid() bool {
	switch c {
	case DebtIntentional, DebtRegression, DebtNeglect, DebtChurn:
		return true
	default:
		return false
	}
}

// String returns the string representation of the debt category.
func (c DebtCategory) String() string {
	return string(c)
}

// DebtItem represents a single item of planning debt.
type DebtItem struct {
	// ID is a unique identifier for this debt item.
	ID string `json:"id" yaml:"id"`

	// ComponentID identifies the spec component (feature/requirement) this debt relates to.
	ComponentID string `json:"component_id" yaml:"component_id"`

	// DriftType is the type of drift that generated this debt item.
	DriftType drift.DriftType `json:"drift_type" yaml:"drift_type"`

	// Category classifies the origin of this debt.
	Category DebtCategory `json:"category" yaml:"category"`

	// Message describes the specific drift issue.
	Message string `json:"message" yaml:"message"`

	// FirstDetected is when this drift was first observed.
	FirstDetected time.Time `json:"first_detected" yaml:"first_detected"`

	// LastDetected is when this drift was most recently observed.
	LastDetected time.Time `json:"last_detected" yaml:"last_detected"`

	// DetectionCount is how many times this drift has been detected.
	DetectionCount int `json:"detection_count" yaml:"detection_count"`

	// DaysPending is how many days this drift has remained unresolved.
	DaysPending int `json:"days_pending" yaml:"days_pending"`

	// IsSticky marks items unresolved for >7 days.
	IsSticky bool `json:"is_sticky" yaml:"is_sticky"`

	// ResolvedAt is when this item was resolved (nil if still active).
	ResolvedAt *time.Time `json:"resolved_at,omitempty" yaml:"resolved_at,omitempty"`
}

// NewDebtItem creates a new debt item from a drift detection.
func NewDebtItem(componentID string, driftType drift.DriftType, message string) *DebtItem {
	now := time.Now()
	return &DebtItem{
		ID:             generateDebtID(componentID, driftType),
		ComponentID:    componentID,
		DriftType:      driftType,
		Category:       DebtNeglect, // Default category, can be updated
		Message:        message,
		FirstDetected:  now,
		LastDetected:   now,
		DetectionCount: 1,
		DaysPending:    0,
		IsSticky:       false,
	}
}

// generateDebtID creates a stable ID for tracking debt items.
func generateDebtID(componentID string, driftType drift.DriftType) string {
	return componentID + "-" + string(driftType)
}

// Update refreshes the debt item with a new detection.
func (d *DebtItem) Update() {
	d.LastDetected = time.Now()
	d.DetectionCount++
	d.DaysPending = int(time.Since(d.FirstDetected).Hours() / 24)
	d.IsSticky = d.DaysPending > 7
}

// Resolve marks the debt item as resolved.
func (d *DebtItem) Resolve() {
	now := time.Now()
	d.ResolvedAt = &now
}

// IsResolved returns true if the debt item has been resolved.
func (d *DebtItem) IsResolved() bool {
	return d.ResolvedAt != nil
}

// SetCategory updates the debt category based on analysis.
func (d *DebtItem) SetCategory(category DebtCategory) {
	d.Category = category
}

// StickyThresholdDays is the number of days after which drift becomes "sticky".
const StickyThresholdDays = 7

// DebtScore represents an aggregated debt score for a component.
type DebtScore struct {
	// ComponentID identifies the spec component.
	ComponentID string `json:"component_id" yaml:"component_id"`

	// Score is the calculated debt score (0-100, higher = more debt).
	Score float64 `json:"score" yaml:"score"`

	// Items are the individual debt items contributing to this score.
	Items []*DebtItem `json:"items" yaml:"items"`

	// StickyCount is the number of sticky items.
	StickyCount int `json:"sticky_count" yaml:"sticky_count"`

	// TotalDaysPending is the sum of all days pending across items.
	TotalDaysPending int `json:"total_days_pending" yaml:"total_days_pending"`
}

// NewDebtScore creates a debt score for a component with its items.
func NewDebtScore(componentID string, items []*DebtItem) *DebtScore {
	score := &DebtScore{
		ComponentID: componentID,
		Items:       items,
	}
	score.Calculate()
	return score
}

// Calculate computes the debt score based on items.
func (s *DebtScore) Calculate() {
	if len(s.Items) == 0 {
		s.Score = 0
		s.StickyCount = 0
		s.TotalDaysPending = 0
		return
	}

	totalDetections := 0
	for _, item := range s.Items {
		if item.IsSticky {
			s.StickyCount++
		}
		s.TotalDaysPending += item.DaysPending
		totalDetections += item.DetectionCount
	}

	// Score formula:
	// - Base: number of items * 10
	// - Sticky penalty: sticky items * 20
	// - Time penalty: total days pending * 0.5
	// - Churn penalty: total detections beyond first * 2
	base := float64(len(s.Items)) * 10
	stickyPenalty := float64(s.StickyCount) * 20
	timePenalty := float64(s.TotalDaysPending) * 0.5
	churnPenalty := float64(totalDetections-len(s.Items)) * 2

	s.Score = base + stickyPenalty + timePenalty + churnPenalty

	// Cap at 100
	if s.Score > 100 {
		s.Score = 100
	}
}

// DebtReport contains a full analysis of planning debt.
type DebtReport struct {
	// GeneratedAt is when this report was created.
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`

	// TotalItems is the count of all debt items.
	TotalItems int `json:"total_items" yaml:"total_items"`

	// StickyItems is the count of sticky debt items.
	StickyItems int `json:"sticky_items" yaml:"sticky_items"`

	// AverageScore is the mean debt score across all components.
	AverageScore float64 `json:"average_score" yaml:"average_score"`

	// MaxScore is the highest component debt score.
	MaxScore float64 `json:"max_score" yaml:"max_score"`

	// Scores contains debt scores by component.
	Scores map[string]*DebtScore `json:"scores" yaml:"scores"`

	// ByCategory groups debt items by category.
	ByCategory map[DebtCategory][]*DebtItem `json:"by_category" yaml:"by_category"`

	// ByDriftType groups debt items by drift type.
	ByDriftType map[drift.DriftType][]*DebtItem `json:"by_drift_type" yaml:"by_drift_type"`

	// StickyDrift lists all sticky debt items.
	StickyDrift []*DebtItem `json:"sticky_drift" yaml:"sticky_drift"`

	// RecentlyResolved lists items resolved in the last 7 days.
	RecentlyResolved []*DebtItem `json:"recently_resolved,omitempty" yaml:"recently_resolved,omitempty"`
}

// NewDebtReport creates an empty debt report.
func NewDebtReport() *DebtReport {
	return &DebtReport{
		GeneratedAt: time.Now(),
		Scores:      make(map[string]*DebtScore),
		ByCategory:  make(map[DebtCategory][]*DebtItem),
		ByDriftType: make(map[drift.DriftType][]*DebtItem),
		StickyDrift: make([]*DebtItem, 0),
	}
}

// AddItem adds a debt item to the report and updates aggregations.
func (r *DebtReport) AddItem(item *DebtItem) {
	r.TotalItems++

	// Add to category grouping
	r.ByCategory[item.Category] = append(r.ByCategory[item.Category], item)

	// Add to drift type grouping
	r.ByDriftType[item.DriftType] = append(r.ByDriftType[item.DriftType], item)

	// Track sticky items
	if item.IsSticky {
		r.StickyItems++
		r.StickyDrift = append(r.StickyDrift, item)
	}

	// Add to component score
	if r.Scores[item.ComponentID] == nil {
		r.Scores[item.ComponentID] = &DebtScore{
			ComponentID: item.ComponentID,
			Items:       make([]*DebtItem, 0),
		}
	}
	r.Scores[item.ComponentID].Items = append(r.Scores[item.ComponentID].Items, item)
}

// Finalize calculates all aggregate scores after items are added.
func (r *DebtReport) Finalize() {
	if len(r.Scores) == 0 {
		r.AverageScore = 0
		r.MaxScore = 0
		return
	}

	totalScore := 0.0
	for _, score := range r.Scores {
		score.Calculate()
		totalScore += score.Score
		if score.Score > r.MaxScore {
			r.MaxScore = score.Score
		}
	}

	r.AverageScore = totalScore / float64(len(r.Scores))
}

// GetHealthLevel returns a health assessment based on the report.
func (r *DebtReport) GetHealthLevel() string {
	if r.AverageScore < 20 {
		return "healthy"
	}
	if r.AverageScore < 50 {
		return "moderate"
	}
	if r.AverageScore < 80 {
		return "concerning"
	}
	return "critical"
}

// GetTopDebtors returns the components with highest debt scores.
func (r *DebtReport) GetTopDebtors(limit int) []*DebtScore {
	if len(r.Scores) == 0 {
		return nil
	}

	// Collect and sort scores
	scores := make([]*DebtScore, 0, len(r.Scores))
	for _, s := range r.Scores {
		scores = append(scores, s)
	}

	// Simple bubble sort for small lists
	for i := range scores {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	if len(scores) <= limit {
		return scores
	}
	return scores[:limit]
}
