package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
	"github.com/spf13/cobra"
)

var (
	forecastDetailed bool
	forecastBurndown bool
	forecastTrend    bool
	forecastJSON     bool
)

var forecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Predict project completion based on current velocity",
	Long: `Forecast provides project completion predictions with optional detailed analysis.

Flags:
  --detailed   Show confidence intervals and all velocity windows
  --burndown   Show burndown chart data
  --trend      Show velocity trend analysis
  --json       Output in JSON format`,
	RunE: runForecast,
}

func runForecast(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	services, err := wiring.BuildAppServices(cwd)
	if err != nil {
		// Non-fatal - we can still try to forecast
	}

	if services == nil || services.Forecast == nil {
		return fmt.Errorf("forecast service not available")
	}

	forecast, err := services.Forecast.GetForecast()
	if err != nil {
		return fmt.Errorf("get forecast: %w", err)
	}
	if forecast == nil {
		return fmt.Errorf("no plan found to forecast")
	}

	if forecastJSON {
		return outputForecastJSON(forecast)
	}

	return outputForecastText(forecast)
}

func outputForecastText(f *analytics.ForecastResult) error {
	fmt.Println("Project Forecast")
	fmt.Println("------------------")
	fmt.Printf("Current Velocity:  %.2f tasks/day\n", f.Velocity)
	fmt.Printf("Remaining Tasks:   %d\n", f.RemainingTasks)
	fmt.Printf("Completed Tasks:   %d\n", f.CompletedTasks)
	fmt.Printf("Total Tasks:       %d\n", f.TotalTasks)
	fmt.Printf("Completion:        %.1f%%\n", f.CompletionRate())

	if f.RemainingTasks == 0 {
		fmt.Println("\nAll tasks complete. Mission accomplished!")
		return nil
	}

	if f.Velocity <= 0 {
		fmt.Println("\nUnable to forecast: No velocity data yet.")
		fmt.Println("Complete some tasks to enable predictions.")
		return nil
	}

	fmt.Printf("\nEstimated Completion: %.0f days\n", f.EstimatedDays)

	// Trend output
	if forecastTrend || forecastDetailed {
		fmt.Println("\nVelocity Trend")
		fmt.Println("--------------")
		fmt.Printf("Direction:  %s\n", formatTrendDirection(f.Trend.Direction))
		fmt.Printf("Slope:      %+.1f%%\n", f.Trend.Slope*100)
		fmt.Printf("Confidence: %.0f%%\n", f.Trend.Confidence*100)

		if len(f.Trend.Windows) > 0 {
			fmt.Println("\nVelocity by Window:")
			for _, w := range f.Trend.Windows {
				fmt.Printf("  %2d-day: %.2f tasks/day (%d completed)\n", w.Days, w.Velocity, w.Count)
			}
		}
	}

	// Detailed output with confidence intervals
	if forecastDetailed {
		fmt.Println("\nConfidence Interval")
		fmt.Println("-------------------")
		ci := f.ConfidenceInterval
		fmt.Printf("Optimistic: %.0f days\n", ci.Low)
		fmt.Printf("Expected:   %.0f days\n", ci.Expected)
		fmt.Printf("Pessimistic: %.0f days\n", ci.High)

		if !f.EstimatedCompletionDate().IsZero() {
			fmt.Printf("\nExpected Completion Date: %s\n", f.EstimatedCompletionDate().Format("2006-01-02"))
		}
	}

	// Burndown output
	if forecastBurndown && len(f.Burndown) > 0 {
		fmt.Println("\nBurndown Chart")
		fmt.Println("--------------")
		for _, point := range f.Burndown {
			dateStr := point.Date.Format("2006-01-02")
			if point.IsProjected() {
				fmt.Printf("%s: %3d (projected)\n", dateStr, point.Projected)
			} else {
				fmt.Printf("%s: %3d\n", dateStr, point.Actual)
			}
		}
	}

	// Status indicator
	fmt.Println()
	if f.IsOnTrack() {
		fmt.Println("Status: ON TRACK")
	} else if f.Trend.Direction == analytics.TrendDecelerating {
		fmt.Println("Status: AT RISK (velocity declining)")
	} else {
		fmt.Println("Status: NEEDS DATA")
	}

	return nil
}

func outputForecastJSON(f *analytics.ForecastResult) error {
	output := map[string]interface{}{
		"velocity":        f.Velocity,
		"remaining_tasks": f.RemainingTasks,
		"completed_tasks": f.CompletedTasks,
		"total_tasks":     f.TotalTasks,
		"completion_rate": f.CompletionRate(),
		"estimated_days":  f.EstimatedDays,
		"is_on_track":     f.IsOnTrack(),
		"trend": map[string]interface{}{
			"direction":  string(f.Trend.Direction),
			"slope":      f.Trend.Slope,
			"confidence": f.Trend.Confidence,
		},
		"confidence_interval": map[string]interface{}{
			"low":      f.ConfidenceInterval.Low,
			"expected": f.ConfidenceInterval.Expected,
			"high":     f.ConfidenceInterval.High,
		},
		"data_points":  f.DataPoints,
		"last_updated": f.LastUpdated,
	}

	if forecastDetailed && !f.EstimatedCompletionDate().IsZero() {
		output["estimated_completion_date"] = f.EstimatedCompletionDate().Format("2006-01-02")
	}

	if forecastTrend || forecastDetailed {
		windows := make([]map[string]interface{}, len(f.Trend.Windows))
		for i, w := range f.Trend.Windows {
			windows[i] = map[string]interface{}{
				"days":     w.Days,
				"velocity": w.Velocity,
				"count":    w.Count,
			}
		}
		output["velocity_windows"] = windows
	}

	if forecastBurndown {
		burndown := make([]map[string]interface{}, len(f.Burndown))
		for i, b := range f.Burndown {
			burndown[i] = map[string]interface{}{
				"date":       b.Date.Format("2006-01-02"),
				"actual":     b.Actual,
				"projected":  b.Projected,
				"is_projected": b.IsProjected(),
			}
		}
		output["burndown"] = burndown
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func formatTrendDirection(d analytics.TrendDirection) string {
	switch d {
	case analytics.TrendAccelerating:
		return "Accelerating (improving)"
	case analytics.TrendDecelerating:
		return "Decelerating (slowing)"
	case analytics.TrendStable:
		return "Stable"
	default:
		return string(d)
	}
}

// RunForecast is the exported RunE function for use as a subcommand
var RunForecast = runForecast

func init() {
	forecastCmd.Flags().BoolVar(&forecastDetailed, "detailed", false, "Show detailed forecast with confidence intervals")
	forecastCmd.Flags().BoolVar(&forecastBurndown, "burndown", false, "Show burndown chart data")
	forecastCmd.Flags().BoolVar(&forecastTrend, "trend", false, "Show velocity trend analysis")
	forecastCmd.Flags().BoolVar(&forecastJSON, "json", false, "Output in JSON format")
	forecastCmd.Hidden = true // Hide from top-level help, available via `status forecast`
	RootCmd.AddCommand(forecastCmd)
}
