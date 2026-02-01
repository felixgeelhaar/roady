package events

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
)

// ExtendedVelocityProjection provides multi-window velocity analysis with trend detection.
type ExtendedVelocityProjection struct {
	mu            sync.RWMutex
	completions   []completionRecord
	windows       []int // Window sizes in days (e.g., 7, 14, 30)
	defaultWindow int
}

// completionRecord tracks when a task was completed and its metadata.
type completionRecord struct {
	TaskID    string
	Timestamp time.Time
	CycleTime time.Duration // Time from start to completion
}

// NewExtendedVelocityProjection creates a projection with configurable windows.
func NewExtendedVelocityProjection(windows ...int) *ExtendedVelocityProjection {
	if len(windows) == 0 {
		windows = []int{7, 14, 30}
	}
	sort.Ints(windows)

	defaultWindow := 7
	if len(windows) > 0 {
		defaultWindow = windows[0]
	}

	return &ExtendedVelocityProjection{
		completions:   make([]completionRecord, 0),
		windows:       windows,
		defaultWindow: defaultWindow,
	}
}

func (p *ExtendedVelocityProjection) Name() string { return "velocity_extended" }

func (p *ExtendedVelocityProjection) Apply(event *BaseEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if event.Type == EventTypeTaskCompleted {
		record := completionRecord{
			TaskID:    getStringMetadata(event.Metadata, "task_id"),
			Timestamp: event.Timestamp,
		}

		// Calculate cycle time if start time is available
		if startedAt, ok := event.Metadata["started_at"].(time.Time); ok {
			record.CycleTime = event.Timestamp.Sub(startedAt)
		}

		p.completions = append(p.completions, record)
	}

	return nil
}

func (p *ExtendedVelocityProjection) Rebuild(events []*BaseEvent) error {
	p.Reset()
	for _, event := range events {
		if err := p.Apply(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *ExtendedVelocityProjection) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completions = make([]completionRecord, 0)
	return nil
}

// GetVelocityWindows returns velocity data for all configured windows.
func (p *ExtendedVelocityProjection) GetVelocityWindows() []analytics.VelocityWindow {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]analytics.VelocityWindow, len(p.windows))
	now := time.Now()

	for i, days := range p.windows {
		cutoff := now.AddDate(0, 0, -days)
		count := 0
		for _, c := range p.completions {
			if c.Timestamp.After(cutoff) {
				count++
			}
		}
		result[i] = analytics.VelocityWindow{
			Days:     days,
			Velocity: float64(count) / float64(days),
			Count:    count,
		}
	}

	return result
}

// GetVelocityTrend analyzes velocity across windows to determine trend direction.
func (p *ExtendedVelocityProjection) GetVelocityTrend() analytics.VelocityTrend {
	windows := p.GetVelocityWindows()

	if len(windows) < 2 {
		return analytics.VelocityTrend{
			Direction:  analytics.TrendStable,
			Slope:      0,
			Confidence: 0,
			Windows:    windows,
		}
	}

	// Calculate trend using linear regression on window velocities
	// Shorter windows represent more recent data
	direction, slope, confidence := p.calculateTrend(windows)

	return analytics.VelocityTrend{
		Direction:  direction,
		Slope:      slope,
		Confidence: confidence,
		Windows:    windows,
	}
}

// calculateTrend determines trend direction from velocity windows.
// Shorter windows (more recent) are weighted more heavily.
func (p *ExtendedVelocityProjection) calculateTrend(windows []analytics.VelocityWindow) (analytics.TrendDirection, float64, float64) {
	if len(windows) < 2 {
		return analytics.TrendStable, 0, 0
	}

	// Compare short-term vs long-term velocity
	shortTerm := windows[0].Velocity // Smallest window (most recent)
	longTerm := windows[len(windows)-1].Velocity // Largest window (includes older data)

	if longTerm == 0 && shortTerm == 0 {
		return analytics.TrendStable, 0, 0
	}

	// Calculate slope as percentage change
	var slope float64
	if longTerm > 0 {
		slope = (shortTerm - longTerm) / longTerm
	} else if shortTerm > 0 {
		slope = 1.0 // Going from 0 to something is maximum acceleration
	}

	// Determine direction based on slope threshold
	var direction analytics.TrendDirection
	threshold := 0.1 // 10% change threshold

	switch {
	case slope > threshold:
		direction = analytics.TrendAccelerating
	case slope < -threshold:
		direction = analytics.TrendDecelerating
	default:
		direction = analytics.TrendStable
	}

	// Calculate confidence based on data consistency
	confidence := p.calculateConfidence(windows)

	return direction, slope, confidence
}

// calculateConfidence determines how reliable the trend analysis is.
func (p *ExtendedVelocityProjection) calculateConfidence(windows []analytics.VelocityWindow) float64 {
	if len(windows) < 2 {
		return 0
	}

	// Confidence is based on:
	// 1. Amount of data (more completions = higher confidence)
	// 2. Consistency across windows (less variance = higher confidence)

	totalCompletions := 0
	velocities := make([]float64, len(windows))

	for i, w := range windows {
		totalCompletions += w.Count
		velocities[i] = w.Velocity
	}

	// Base confidence on data volume (logarithmic scale)
	dataConfidence := math.Min(math.Log10(float64(totalCompletions+1))/2, 1.0)

	// Calculate variance in velocities
	mean := 0.0
	for _, v := range velocities {
		mean += v
	}
	mean /= float64(len(velocities))

	variance := 0.0
	for _, v := range velocities {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(velocities))

	// Consistency confidence (lower variance = higher confidence)
	var consistencyConfidence float64
	if mean > 0 {
		cv := math.Sqrt(variance) / mean // Coefficient of variation
		consistencyConfidence = math.Max(0, 1-cv)
	} else {
		consistencyConfidence = 0.5 // Neutral if no mean
	}

	// Combine confidence factors
	return (dataConfidence + consistencyConfidence) / 2
}

// GetVelocityStats returns statistical summary of velocity.
func (p *ExtendedVelocityProjection) GetVelocityStats() analytics.VelocityStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.completions) == 0 {
		return analytics.VelocityStats{}
	}

	// Calculate daily completion counts
	dailyCounts := p.getDailyCounts()
	if len(dailyCounts) == 0 {
		return analytics.VelocityStats{Samples: 0}
	}

	// Sort for median calculation
	sorted := make([]float64, len(dailyCounts))
	copy(sorted, dailyCounts)
	sort.Float64s(sorted)

	// Calculate statistics
	var sum float64
	min := sorted[0]
	max := sorted[len(sorted)-1]

	for _, v := range sorted {
		sum += v
	}
	mean := sum / float64(len(sorted))

	// Median
	var median float64
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		median = (sorted[mid-1] + sorted[mid]) / 2
	} else {
		median = sorted[mid]
	}

	// Standard deviation
	var variance float64
	for _, v := range sorted {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(sorted))
	stdDev := math.Sqrt(variance)

	return analytics.VelocityStats{
		Mean:    mean,
		Median:  median,
		StdDev:  stdDev,
		Min:     min,
		Max:     max,
		Samples: len(sorted),
	}
}

// getDailyCounts returns completion counts per day.
func (p *ExtendedVelocityProjection) getDailyCounts() []float64 {
	if len(p.completions) == 0 {
		return nil
	}

	// Group completions by day
	dayCount := make(map[string]int)
	for _, c := range p.completions {
		day := c.Timestamp.Format("2006-01-02")
		dayCount[day]++
	}

	result := make([]float64, 0, len(dayCount))
	for _, count := range dayCount {
		result = append(result, float64(count))
	}

	return result
}

// GenerateBurndown generates burndown data for forecasting.
func (p *ExtendedVelocityProjection) GenerateBurndown(totalTasks, remainingTasks int, projectedDays int) []analytics.BurndownPoint {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]analytics.BurndownPoint, 0)
	now := time.Now()

	// Add historical data points (actual burndown)
	historicalPoints := p.getHistoricalBurndown(totalTasks)
	result = append(result, historicalPoints...)

	// Add projected data points
	if remainingTasks > 0 && projectedDays > 0 {
		trend := p.GetVelocityTrend()
		velocity := trend.Windows[0].Velocity // Use short-term velocity

		if velocity > 0 {
			remaining := float64(remainingTasks)
			for day := 1; day <= projectedDays && remaining > 0; day++ {
				remaining -= velocity
				if remaining < 0 {
					remaining = 0
				}
				result = append(result, analytics.BurndownPoint{
					Date:      now.AddDate(0, 0, day),
					Actual:    0,
					Projected: int(math.Ceil(remaining)),
				})
			}
		}
	}

	return result
}

// getHistoricalBurndown calculates historical burndown from completion events.
func (p *ExtendedVelocityProjection) getHistoricalBurndown(totalTasks int) []analytics.BurndownPoint {
	if len(p.completions) == 0 {
		return nil
	}

	// Sort completions by time
	sorted := make([]completionRecord, len(p.completions))
	copy(sorted, p.completions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Generate daily snapshots
	result := make([]analytics.BurndownPoint, 0)
	remaining := totalTasks
	currentDay := ""

	for _, c := range sorted {
		day := c.Timestamp.Format("2006-01-02")
		if day != currentDay {
			if currentDay != "" {
				// Save previous day's state
				t, _ := time.Parse("2006-01-02", currentDay)
				result = append(result, analytics.BurndownPoint{
					Date:   t,
					Actual: remaining,
				})
			}
			currentDay = day
		}
		remaining--
		if remaining < 0 {
			remaining = 0
		}
	}

	// Add final day
	if currentDay != "" {
		t, _ := time.Parse("2006-01-02", currentDay)
		result = append(result, analytics.BurndownPoint{
			Date:   t,
			Actual: remaining,
		})
	}

	return result
}

// GetCompletionCount returns the total number of completions.
func (p *ExtendedVelocityProjection) GetCompletionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.completions)
}

// GetCompletionsInWindow returns completions within the specified days.
func (p *ExtendedVelocityProjection) GetCompletionsInWindow(days int) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	count := 0
	for _, c := range p.completions {
		if c.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}
