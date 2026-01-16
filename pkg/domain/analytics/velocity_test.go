package analytics

import (
	"testing"
	"time"
)

func TestVelocityTrend_IsPositive(t *testing.T) {
	tests := []struct {
		name      string
		direction TrendDirection
		want      bool
	}{
		{"accelerating", TrendAccelerating, true},
		{"decelerating", TrendDecelerating, false},
		{"stable", TrendStable, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trend := VelocityTrend{Direction: tt.direction}
			if got := trend.IsPositive(); got != tt.want {
				t.Errorf("VelocityTrend.IsPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVelocityTrend_IsNegative(t *testing.T) {
	tests := []struct {
		name      string
		direction TrendDirection
		want      bool
	}{
		{"accelerating", TrendAccelerating, false},
		{"decelerating", TrendDecelerating, true},
		{"stable", TrendStable, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trend := VelocityTrend{Direction: tt.direction}
			if got := trend.IsNegative(); got != tt.want {
				t.Errorf("VelocityTrend.IsNegative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfidenceInterval_Range(t *testing.T) {
	ci := ConfidenceInterval{Low: 5.0, Expected: 10.0, High: 20.0}
	want := 15.0
	if got := ci.Range(); got != want {
		t.Errorf("ConfidenceInterval.Range() = %v, want %v", got, want)
	}
}

func TestConfidenceInterval_IsNarrow(t *testing.T) {
	tests := []struct {
		name string
		ci   ConfidenceInterval
		want bool
	}{
		{
			name: "narrow interval",
			ci:   ConfidenceInterval{Low: 8.0, Expected: 10.0, High: 12.0},
			want: true,
		},
		{
			name: "wide interval",
			ci:   ConfidenceInterval{Low: 5.0, Expected: 10.0, High: 20.0},
			want: false,
		},
		{
			name: "zero expected",
			ci:   ConfidenceInterval{Low: 0, Expected: 0, High: 5.0},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ci.IsNarrow(); got != tt.want {
				t.Errorf("ConfidenceInterval.IsNarrow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBurndownPoint_IsProjected(t *testing.T) {
	tests := []struct {
		name string
		bp   BurndownPoint
		want bool
	}{
		{
			name: "actual data",
			bp:   BurndownPoint{Actual: 10, Projected: 0},
			want: false,
		},
		{
			name: "projected data",
			bp:   BurndownPoint{Actual: 0, Projected: 10},
			want: true,
		},
		{
			name: "both values",
			bp:   BurndownPoint{Actual: 5, Projected: 10},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bp.IsProjected(); got != tt.want {
				t.Errorf("BurndownPoint.IsProjected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskCycleTime_Efficiency(t *testing.T) {
	tests := []struct {
		name      string
		totalTime time.Duration
		workTime  time.Duration
		want      float64
	}{
		{
			name:      "half efficiency",
			totalTime: 10 * time.Hour,
			workTime:  5 * time.Hour,
			want:      0.5,
		},
		{
			name:      "full efficiency",
			totalTime: 8 * time.Hour,
			workTime:  8 * time.Hour,
			want:      1.0,
		},
		{
			name:      "zero total",
			totalTime: 0,
			workTime:  5 * time.Hour,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := TaskCycleTime{TotalTime: tt.totalTime, WorkTime: tt.workTime}
			if got := tc.Efficiency(); got != tt.want {
				t.Errorf("TaskCycleTime.Efficiency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForecastResult_CompletionRate(t *testing.T) {
	tests := []struct {
		name      string
		completed int
		total     int
		want      float64
	}{
		{"50% complete", 5, 10, 50.0},
		{"100% complete", 10, 10, 100.0},
		{"0% complete", 0, 10, 0.0},
		{"no tasks", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ForecastResult{CompletedTasks: tt.completed, TotalTasks: tt.total}
			if got := f.CompletionRate(); got != tt.want {
				t.Errorf("ForecastResult.CompletionRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForecastResult_IsOnTrack(t *testing.T) {
	tests := []struct {
		name     string
		trend    TrendDirection
		velocity float64
		want     bool
	}{
		{"accelerating with velocity", TrendAccelerating, 2.0, true},
		{"stable with velocity", TrendStable, 1.5, true},
		{"decelerating", TrendDecelerating, 2.0, false},
		{"zero velocity", TrendAccelerating, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ForecastResult{
				Trend:    VelocityTrend{Direction: tt.trend},
				Velocity: tt.velocity,
			}
			if got := f.IsOnTrack(); got != tt.want {
				t.Errorf("ForecastResult.IsOnTrack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForecastResult_EstimatedCompletionDate(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name           string
		velocity       float64
		remainingTasks int
		estimatedDays  float64
		wantZero       bool
	}{
		{
			name:           "valid forecast",
			velocity:       2.0,
			remainingTasks: 10,
			estimatedDays:  5.0,
			wantZero:       false,
		},
		{
			name:           "zero velocity",
			velocity:       0,
			remainingTasks: 10,
			estimatedDays:  5.0,
			wantZero:       true,
		},
		{
			name:           "no remaining tasks",
			velocity:       2.0,
			remainingTasks: 0,
			estimatedDays:  0,
			wantZero:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ForecastResult{
				Velocity:       tt.velocity,
				RemainingTasks: tt.remainingTasks,
				EstimatedDays:  tt.estimatedDays,
				LastUpdated:    now,
			}
			got := f.EstimatedCompletionDate()
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("EstimatedCompletionDate() should be zero, got %v", got)
				}
			} else {
				if got.IsZero() {
					t.Error("EstimatedCompletionDate() should not be zero")
				}
			}
		})
	}
}

func TestVelocityStats_Variability(t *testing.T) {
	tests := []struct {
		name   string
		mean   float64
		stdDev float64
		want   float64
	}{
		{"typical", 10.0, 2.0, 0.2},
		{"zero mean", 0, 2.0, 0},
		{"high variability", 10.0, 5.0, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := VelocityStats{Mean: tt.mean, StdDev: tt.stdDev}
			if got := vs.Variability(); got != tt.want {
				t.Errorf("VelocityStats.Variability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVelocityStats_IsConsistent(t *testing.T) {
	tests := []struct {
		name   string
		mean   float64
		stdDev float64
		want   bool
	}{
		{"consistent", 10.0, 2.0, true},
		{"inconsistent", 10.0, 5.0, false},
		{"borderline", 10.0, 3.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := VelocityStats{Mean: tt.mean, StdDev: tt.stdDev}
			if got := vs.IsConsistent(); got != tt.want {
				t.Errorf("VelocityStats.IsConsistent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVelocityWindow(t *testing.T) {
	window := VelocityWindow{
		Days:     7,
		Velocity: 2.5,
		Count:    17,
	}

	if window.Days != 7 {
		t.Errorf("Expected Days 7, got %d", window.Days)
	}
	if window.Velocity != 2.5 {
		t.Errorf("Expected Velocity 2.5, got %f", window.Velocity)
	}
	if window.Count != 17 {
		t.Errorf("Expected Count 17, got %d", window.Count)
	}
}

func TestTrendDirectionConstants(t *testing.T) {
	if TrendAccelerating != "accelerating" {
		t.Error("TrendAccelerating should be 'accelerating'")
	}
	if TrendDecelerating != "decelerating" {
		t.Error("TrendDecelerating should be 'decelerating'")
	}
	if TrendStable != "stable" {
		t.Error("TrendStable should be 'stable'")
	}
}
