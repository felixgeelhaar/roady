package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
)

func resetForecastFlags() {
	forecastDetailed = false
	forecastBurndown = false
	forecastTrend = false
	forecastJSON = false
}

func TestOutputForecastJSON_WithDetails(t *testing.T) {
	defer resetForecastFlags()

	forecastDetailed = true
	forecastBurndown = true
	forecastTrend = true

	lastUpdated := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)
	forecast := &analytics.ForecastResult{
		RemainingTasks: 5,
		CompletedTasks: 10,
		TotalTasks:     15,
		Velocity:       2.5,
		EstimatedDays:  4,
		ConfidenceInterval: analytics.ConfidenceInterval{
			Low:      3,
			Expected: 4,
			High:     6,
		},
		Trend: analytics.VelocityTrend{
			Direction:  analytics.TrendAccelerating,
			Slope:      0.12,
			Confidence: 0.8,
			Windows: []analytics.VelocityWindow{
				{Days: 7, Velocity: 2.0, Count: 14},
				{Days: 14, Velocity: 2.5, Count: 35},
			},
		},
		Burndown: []analytics.BurndownPoint{
			{Date: lastUpdated.AddDate(0, 0, -1), Actual: 6, Projected: 0},
			{Date: lastUpdated.AddDate(0, 0, 1), Actual: 0, Projected: 4},
		},
		LastUpdated: lastUpdated,
		DataPoints:  20,
	}

	output := captureStdout(t, func() {
		if err := outputForecastJSON(forecast); err != nil {
			t.Fatalf("outputForecastJSON failed: %v", err)
		}
	})

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode json: %v\n%s", err, output)
	}

	if payload["estimated_completion_date"] != "2026-01-23" {
		t.Fatalf("unexpected estimated_completion_date: %v", payload["estimated_completion_date"])
	}

	windows, ok := payload["velocity_windows"].([]interface{})
	if !ok || len(windows) != 2 {
		t.Fatalf("expected velocity_windows length 2, got: %v", payload["velocity_windows"])
	}

	burndown, ok := payload["burndown"].([]interface{})
	if !ok || len(burndown) != 2 {
		t.Fatalf("expected burndown length 2, got: %v", payload["burndown"])
	}
	if !strings.Contains(output, "\"is_projected\": true") {
		t.Fatal("expected projected burndown point to be marked")
	}
}

func TestOutputForecastText_AllTasksComplete(t *testing.T) {
	defer resetForecastFlags()

	forecast := &analytics.ForecastResult{
		RemainingTasks: 0,
		CompletedTasks: 10,
		TotalTasks:     10,
		Velocity:       2.0,
	}

	output := captureStdout(t, func() {
		if err := outputForecastText(forecast); err != nil {
			t.Fatalf("outputForecastText failed: %v", err)
		}
	})

	if !strings.Contains(output, "All tasks complete") {
		t.Fatalf("expected completion message, got:\n%s", output)
	}
}

func TestOutputForecastText_NoVelocity(t *testing.T) {
	defer resetForecastFlags()

	forecast := &analytics.ForecastResult{
		RemainingTasks: 5,
		CompletedTasks: 0,
		TotalTasks:     5,
		Velocity:       0,
	}

	output := captureStdout(t, func() {
		if err := outputForecastText(forecast); err != nil {
			t.Fatalf("outputForecastText failed: %v", err)
		}
	})

	if !strings.Contains(output, "Unable to forecast") {
		t.Fatalf("expected no-velocity message, got:\n%s", output)
	}
}

func TestOutputForecastText_WithDetails(t *testing.T) {
	defer resetForecastFlags()

	forecastDetailed = true
	forecastBurndown = true
	forecastTrend = true

	lastUpdated := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)
	forecast := &analytics.ForecastResult{
		RemainingTasks: 3,
		CompletedTasks: 7,
		TotalTasks:     10,
		Velocity:       1.5,
		EstimatedDays:  2,
		ConfidenceInterval: analytics.ConfidenceInterval{
			Low:      1,
			Expected: 2,
			High:     3,
		},
		Trend: analytics.VelocityTrend{
			Direction:  analytics.TrendStable,
			Slope:      0.01,
			Confidence: 0.6,
			Windows: []analytics.VelocityWindow{
				{Days: 7, Velocity: 1.2, Count: 8},
			},
		},
		Burndown: []analytics.BurndownPoint{
			{Date: lastUpdated.AddDate(0, 0, -1), Actual: 4},
			{Date: lastUpdated.AddDate(0, 0, 1), Projected: 2},
		},
		LastUpdated: lastUpdated,
	}

	output := captureStdout(t, func() {
		if err := outputForecastText(forecast); err != nil {
			t.Fatalf("outputForecastText failed: %v", err)
		}
	})

	for _, needle := range []string{
		"Velocity Trend",
		"Confidence Interval",
		"Burndown Chart",
		"Status:",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("expected %q in output:\n%s", needle, output)
		}
	}
}

func TestFormatTrendDirection(t *testing.T) {
	tests := []struct {
		name     string
		input    analytics.TrendDirection
		expected string
	}{
		{"accelerating", analytics.TrendAccelerating, "Accelerating (improving)"},
		{"decelerating", analytics.TrendDecelerating, "Decelerating (slowing)"},
		{"stable", analytics.TrendStable, "Stable"},
		{"unknown", analytics.TrendDirection("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTrendDirection(tt.input); got != tt.expected {
				t.Fatalf("formatTrendDirection(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}
