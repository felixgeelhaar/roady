package events

import (
	"sync"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/debt"
	"github.com/felixgeelhaar/roady/pkg/domain/drift"
)

// DriftHistoryProjection tracks drift detection events over time for debt analysis.
type DriftHistoryProjection struct {
	mu       sync.RWMutex
	history  []DriftSnapshot
	items    map[string]*debt.DebtItem // Active debt items by ID
	resolved map[string]*debt.DebtItem // Resolved debt items by ID
}

// DriftSnapshot represents a point-in-time drift detection.
type DriftSnapshot struct {
	Timestamp   time.Time      `json:"timestamp"`
	IssueCount  int            `json:"issue_count"`
	ComponentID string         `json:"component_id,omitempty"`
	DriftType   drift.DriftType `json:"drift_type,omitempty"`
	Category    drift.DriftCategory `json:"category,omitempty"`
	Message     string         `json:"message,omitempty"`
	EventID     string         `json:"event_id"`
}

// NewDriftHistoryProjection creates a new drift history projection.
func NewDriftHistoryProjection() *DriftHistoryProjection {
	return &DriftHistoryProjection{
		history:  make([]DriftSnapshot, 0),
		items:    make(map[string]*debt.DebtItem),
		resolved: make(map[string]*debt.DebtItem),
	}
}

func (p *DriftHistoryProjection) Name() string { return "drift_history" }

func (p *DriftHistoryProjection) Apply(event *BaseEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch event.Type {
	case EventTypeDriftDetected:
		return p.applyDriftDetected(event)
	case EventTypeDriftAccepted:
		return p.applyDriftAccepted(event)
	case EventTypeDriftResolved:
		return p.applyDriftResolved(event)
	}

	return nil
}

func (p *DriftHistoryProjection) applyDriftDetected(event *BaseEvent) error {
	// Extract drift information from event metadata
	componentID := getStringMetadata(event.Metadata, "component_id")
	driftTypeStr := getStringMetadata(event.Metadata, "drift_type")
	categoryStr := getStringMetadata(event.Metadata, "category")
	message := getStringMetadata(event.Metadata, "message")
	issueCount := getIntMetadata(event.Metadata, "issue_count")

	// Record snapshot
	snapshot := DriftSnapshot{
		Timestamp:   event.Timestamp,
		IssueCount:  issueCount,
		ComponentID: componentID,
		DriftType:   drift.DriftType(driftTypeStr),
		Category:    drift.DriftCategory(categoryStr),
		Message:     message,
		EventID:     event.ID,
	}
	p.history = append(p.history, snapshot)

	// Update or create debt item if component-specific
	if componentID != "" {
		itemID := generateDebtItemID(componentID, drift.DriftType(driftTypeStr))
		if existing, ok := p.items[itemID]; ok {
			existing.Update()
			p.categorizeDebtItem(existing)
		} else {
			item := debt.NewDebtItem(componentID, drift.DriftType(driftTypeStr), message)
			// Check if this is a regression (was previously resolved)
			if _, wasResolved := p.resolved[itemID]; wasResolved {
				item.SetCategory(debt.DebtRegression)
			}
			p.items[itemID] = item
		}
	}

	return nil
}

func (p *DriftHistoryProjection) applyDriftAccepted(event *BaseEvent) error {
	componentID := getStringMetadata(event.Metadata, "component_id")
	driftTypeStr := getStringMetadata(event.Metadata, "drift_type")

	if componentID != "" {
		itemID := generateDebtItemID(componentID, drift.DriftType(driftTypeStr))
		if item, ok := p.items[itemID]; ok {
			// Mark as intentional debt
			item.SetCategory(debt.DebtIntentional)
		}
	}

	return nil
}

func (p *DriftHistoryProjection) applyDriftResolved(event *BaseEvent) error {
	componentID := getStringMetadata(event.Metadata, "component_id")
	driftTypeStr := getStringMetadata(event.Metadata, "drift_type")

	if componentID != "" {
		itemID := generateDebtItemID(componentID, drift.DriftType(driftTypeStr))
		if item, ok := p.items[itemID]; ok {
			item.Resolve()
			p.resolved[itemID] = item
			delete(p.items, itemID)
		}
	}

	return nil
}

// categorizeDebtItem assigns a category based on detection patterns.
func (p *DriftHistoryProjection) categorizeDebtItem(item *debt.DebtItem) {
	if item.DetectionCount > 3 && item.DaysPending < 7 {
		// Detected multiple times but recently - likely churn
		item.SetCategory(debt.DebtChurn)
	} else if item.DaysPending > 14 {
		// Long-standing issue - neglect
		item.SetCategory(debt.DebtNeglect)
	}
	// Check if it was previously resolved (regression)
	if _, wasResolved := p.resolved[item.ID]; wasResolved {
		item.SetCategory(debt.DebtRegression)
	}
}

func generateDebtItemID(componentID string, driftType drift.DriftType) string {
	return componentID + "-" + string(driftType)
}

func (p *DriftHistoryProjection) Rebuild(events []*BaseEvent) error {
	p.Reset()
	for _, event := range events {
		if err := p.Apply(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *DriftHistoryProjection) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.history = make([]DriftSnapshot, 0)
	p.items = make(map[string]*debt.DebtItem)
	p.resolved = make(map[string]*debt.DebtItem)
	return nil
}

// GetActiveDebtItems returns all currently active debt items.
func (p *DriftHistoryProjection) GetActiveDebtItems() []*debt.DebtItem {
	p.mu.RLock()
	defer p.mu.RUnlock()

	items := make([]*debt.DebtItem, 0, len(p.items))
	for _, item := range p.items {
		items = append(items, item)
	}
	return items
}

// GetStickyDebtItems returns debt items that have been pending for more than 7 days.
func (p *DriftHistoryProjection) GetStickyDebtItems() []*debt.DebtItem {
	p.mu.RLock()
	defer p.mu.RUnlock()

	items := make([]*debt.DebtItem, 0)
	for _, item := range p.items {
		if item.IsSticky {
			items = append(items, item)
		}
	}
	return items
}

// GetResolvedDebtItems returns recently resolved debt items.
func (p *DriftHistoryProjection) GetResolvedDebtItems() []*debt.DebtItem {
	p.mu.RLock()
	defer p.mu.RUnlock()

	items := make([]*debt.DebtItem, 0, len(p.resolved))
	for _, item := range p.resolved {
		items = append(items, item)
	}
	return items
}

// GetDebtByComponent groups debt items by their component ID.
func (p *DriftHistoryProjection) GetDebtByComponent() map[string][]*debt.DebtItem {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string][]*debt.DebtItem)
	for _, item := range p.items {
		result[item.ComponentID] = append(result[item.ComponentID], item)
	}
	return result
}

// GetDebtByCategory groups debt items by their category.
func (p *DriftHistoryProjection) GetDebtByCategory() map[debt.DebtCategory][]*debt.DebtItem {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[debt.DebtCategory][]*debt.DebtItem)
	for _, item := range p.items {
		result[item.Category] = append(result[item.Category], item)
	}
	return result
}

// GetDriftHistory returns the historical drift snapshots.
func (p *DriftHistoryProjection) GetDriftHistory() []DriftSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]DriftSnapshot, len(p.history))
	copy(result, p.history)
	return result
}

// GetDriftHistoryInWindow returns drift snapshots within the specified days.
func (p *DriftHistoryProjection) GetDriftHistoryInWindow(days int) []DriftSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	result := make([]DriftSnapshot, 0)
	for _, snapshot := range p.history {
		if snapshot.Timestamp.After(cutoff) {
			result = append(result, snapshot)
		}
	}
	return result
}

// GetDebtReport generates a comprehensive debt report.
func (p *DriftHistoryProjection) GetDebtReport() *debt.DebtReport {
	p.mu.RLock()
	defer p.mu.RUnlock()

	report := debt.NewDebtReport()

	for _, item := range p.items {
		report.AddItem(item)
	}

	// Add recently resolved items (within 7 days)
	cutoff := time.Now().AddDate(0, 0, -7)
	for _, item := range p.resolved {
		if item.ResolvedAt != nil && item.ResolvedAt.After(cutoff) {
			report.RecentlyResolved = append(report.RecentlyResolved, item)
		}
	}

	report.Finalize()
	return report
}

// GetDriftTrend analyzes drift patterns over time.
func (p *DriftHistoryProjection) GetDriftTrend(windowDays int) DriftTrend {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	halfWindow := windowDays / 2

	// Split history into two periods
	firstHalf := now.AddDate(0, 0, -windowDays)
	midPoint := now.AddDate(0, 0, -halfWindow)

	firstCount := 0
	secondCount := 0

	for _, snapshot := range p.history {
		if snapshot.Timestamp.After(firstHalf) && snapshot.Timestamp.Before(midPoint) {
			firstCount += snapshot.IssueCount
		} else if snapshot.Timestamp.After(midPoint) {
			secondCount += snapshot.IssueCount
		}
	}

	var direction string
	var change float64

	if firstCount > 0 {
		change = float64(secondCount-firstCount) / float64(firstCount)
	} else if secondCount > 0 {
		change = 1.0 // New drift appearing
	}

	switch {
	case change > 0.1:
		direction = "increasing"
	case change < -0.1:
		direction = "decreasing"
	default:
		direction = "stable"
	}

	return DriftTrend{
		Direction:       direction,
		Change:          change,
		PreviousPeriod:  firstCount,
		CurrentPeriod:   secondCount,
		WindowDays:      windowDays,
		ActiveDebtCount: len(p.items),
		StickyCount:     len(p.GetStickyDebtItems()),
	}
}

// DriftTrend represents the trend analysis of drift patterns.
type DriftTrend struct {
	Direction       string  `json:"direction"` // increasing, decreasing, stable
	Change          float64 `json:"change"`    // Percentage change
	PreviousPeriod  int     `json:"previous_period"`
	CurrentPeriod   int     `json:"current_period"`
	WindowDays      int     `json:"window_days"`
	ActiveDebtCount int     `json:"active_debt_count"`
	StickyCount     int     `json:"sticky_count"`
}

// helper function to get int from metadata
func getIntMetadata(metadata map[string]any, key string) int {
	if v, ok := metadata[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}
