// Package analytics provides velocity and forecasting analytics for project planning.
package analytics

import (
	"time"
)

// TrendDirection indicates the direction of velocity change over time.
type TrendDirection string

const (
	// TrendAccelerating indicates velocity is increasing.
	TrendAccelerating TrendDirection = "accelerating"
	// TrendDecelerating indicates velocity is decreasing.
	TrendDecelerating TrendDirection = "decelerating"
	// TrendStable indicates velocity is relatively constant.
	TrendStable TrendDirection = "stable"
)

// VelocityWindow represents velocity calculated over a specific time window.
type VelocityWindow struct {
	Days     int     // Number of days in this window (e.g., 7, 14, 30)
	Velocity float64 // Tasks completed per day in this window
	Count    int     // Number of tasks completed in this window
}

// VelocityTrend captures the trend analysis of velocity over multiple time windows.
type VelocityTrend struct {
	Direction  TrendDirection   // Overall trend direction
	Slope      float64          // Rate of change (positive = accelerating)
	Confidence float64          // Confidence level of the trend (0.0-1.0)
	Windows    []VelocityWindow // Velocity data per time window
}

// IsPositive returns true if the trend indicates improving velocity.
func (t VelocityTrend) IsPositive() bool {
	return t.Direction == TrendAccelerating
}

// IsNegative returns true if the trend indicates declining velocity.
func (t VelocityTrend) IsNegative() bool {
	return t.Direction == TrendDecelerating
}

// ConfidenceInterval represents low/expected/high estimates for forecasting.
type ConfidenceInterval struct {
	Low      float64 // Pessimistic estimate (e.g., 10th percentile)
	Expected float64 // Most likely estimate (e.g., 50th percentile)
	High     float64 // Optimistic estimate (e.g., 90th percentile)
}

// Range returns the difference between high and low estimates.
func (ci ConfidenceInterval) Range() float64 {
	return ci.High - ci.Low
}

// IsNarrow returns true if the confidence interval is relatively tight.
func (ci ConfidenceInterval) IsNarrow() bool {
	if ci.Expected == 0 {
		return false
	}
	return ci.Range()/ci.Expected < 0.5 // Less than 50% variance
}

// BurndownPoint represents a single point on a burndown chart.
type BurndownPoint struct {
	Date      time.Time // Date of this data point
	Actual    int       // Actual remaining tasks (historical data)
	Projected int       // Projected remaining tasks (forecast)
}

// IsProjected returns true if this point is a forecast rather than actual data.
func (bp BurndownPoint) IsProjected() bool {
	return bp.Actual == 0 && bp.Projected > 0
}

// TaskCycleTime tracks how long tasks spend in each status.
type TaskCycleTime struct {
	TaskID       string        // ID of the task
	TotalTime    time.Duration // Total time from pending to done
	WaitTime     time.Duration // Time spent in pending/blocked
	WorkTime     time.Duration // Time spent in progress
	StartedAt    time.Time     // When task moved to in_progress
	CompletedAt  time.Time     // When task moved to done
}

// Efficiency returns the ratio of work time to total time.
func (tc TaskCycleTime) Efficiency() float64 {
	if tc.TotalTime == 0 {
		return 0
	}
	return float64(tc.WorkTime) / float64(tc.TotalTime)
}

// ForecastResult contains comprehensive forecasting data.
type ForecastResult struct {
	// Core metrics
	RemainingTasks int     // Number of tasks remaining
	CompletedTasks int     // Number of tasks completed
	TotalTasks     int     // Total tasks in plan
	Velocity       float64 // Current velocity (tasks/day)

	// Time estimates
	EstimatedDays      float64            // Expected days to completion
	ConfidenceInterval ConfidenceInterval // Low/Expected/High day estimates

	// Trend analysis
	Trend VelocityTrend // Velocity trend over time windows

	// Burndown data
	Burndown []BurndownPoint // Historical and projected burndown

	// Additional context
	LastUpdated time.Time // When this forecast was generated
	DataPoints  int       // Number of data points used for calculation
}

// CompletionRate returns the percentage of tasks completed.
func (f ForecastResult) CompletionRate() float64 {
	if f.TotalTasks == 0 {
		return 0
	}
	return float64(f.CompletedTasks) / float64(f.TotalTasks) * 100
}

// IsOnTrack returns true if current velocity supports estimated completion.
func (f ForecastResult) IsOnTrack() bool {
	return f.Trend.Direction != TrendDecelerating && f.Velocity > 0
}

// EstimatedCompletionDate returns the projected completion date.
func (f ForecastResult) EstimatedCompletionDate() time.Time {
	if f.Velocity == 0 || f.RemainingTasks == 0 {
		return time.Time{}
	}
	days := time.Duration(f.EstimatedDays * 24 * float64(time.Hour))
	return f.LastUpdated.Add(days)
}

// VelocityStats holds statistical summary of velocity data.
type VelocityStats struct {
	Mean     float64 // Average velocity
	Median   float64 // Median velocity
	StdDev   float64 // Standard deviation
	Min      float64 // Minimum observed velocity
	Max      float64 // Maximum observed velocity
	Samples  int     // Number of samples
}

// Variability returns the coefficient of variation (StdDev/Mean).
func (vs VelocityStats) Variability() float64 {
	if vs.Mean == 0 {
		return 0
	}
	return vs.StdDev / vs.Mean
}

// IsConsistent returns true if velocity is relatively stable.
func (vs VelocityStats) IsConsistent() bool {
	return vs.Variability() < 0.3 // Less than 30% coefficient of variation
}
