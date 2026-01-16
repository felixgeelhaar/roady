package application

import (
	"math"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/analytics"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// ForecastService provides enhanced project forecasting with trend analysis.
type ForecastService struct {
	projection *events.ExtendedVelocityProjection
	repo       domain.WorkspaceRepository
}

// NewForecastService creates a new forecast service.
func NewForecastService(
	projection *events.ExtendedVelocityProjection,
	repo domain.WorkspaceRepository,
) *ForecastService {
	return &ForecastService{
		projection: projection,
		repo:       repo,
	}
}

// GetForecast returns a comprehensive forecast for the current project.
func (s *ForecastService) GetForecast() (*analytics.ForecastResult, error) {
	plan, err := s.repo.LoadPlan()
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil
	}

	state, _ := s.repo.LoadState()

	// Count task statuses
	totalTasks := len(plan.Tasks)
	completedTasks := 0
	remainingTasks := 0

	for _, t := range plan.Tasks {
		status := planning.StatusPending
		if state != nil {
			if res, ok := state.TaskStates[t.ID]; ok {
				status = res.Status
			}
		}
		if status == planning.StatusVerified || status == planning.StatusDone {
			completedTasks++
		} else {
			remainingTasks++
		}
	}

	// Get velocity trend
	trend := s.projection.GetVelocityTrend()

	// Use short-term velocity for forecasting (most recent window)
	velocity := 0.0
	if len(trend.Windows) > 0 {
		velocity = trend.Windows[0].Velocity
	}

	// Calculate estimated days
	var estimatedDays float64
	if velocity > 0 && remainingTasks > 0 {
		estimatedDays = float64(remainingTasks) / velocity
	}

	// Calculate confidence interval
	ci := s.calculateConfidenceInterval(remainingTasks, velocity, trend)

	// Generate burndown data
	burndown := s.projection.GenerateBurndown(totalTasks, remainingTasks, int(math.Ceil(estimatedDays*1.5)))

	return &analytics.ForecastResult{
		RemainingTasks:     remainingTasks,
		CompletedTasks:     completedTasks,
		TotalTasks:         totalTasks,
		Velocity:           velocity,
		EstimatedDays:      estimatedDays,
		ConfidenceInterval: ci,
		Trend:              trend,
		Burndown:           burndown,
		LastUpdated:        time.Now(),
		DataPoints:         s.projection.GetCompletionCount(),
	}, nil
}

// calculateConfidenceInterval computes low/expected/high estimates.
func (s *ForecastService) calculateConfidenceInterval(
	remaining int,
	velocity float64,
	trend analytics.VelocityTrend,
) analytics.ConfidenceInterval {
	if velocity <= 0 || remaining == 0 {
		return analytics.ConfidenceInterval{}
	}

	expected := float64(remaining) / velocity

	// Adjust based on trend
	var low, high float64
	switch trend.Direction {
	case analytics.TrendAccelerating:
		// Optimistic: velocity might increase
		low = expected * 0.7
		high = expected * 1.2
	case analytics.TrendDecelerating:
		// Pessimistic: velocity might decrease further
		low = expected * 0.9
		high = expected * 1.8
	default:
		// Stable: moderate variance
		low = expected * 0.8
		high = expected * 1.3
	}

	// Adjust based on confidence level
	if trend.Confidence < 0.5 {
		// Low confidence: widen the interval
		low *= 0.8
		high *= 1.5
	}

	return analytics.ConfidenceInterval{
		Low:      low,
		Expected: expected,
		High:     high,
	}
}

// GetVelocityTrend returns the current velocity trend analysis.
func (s *ForecastService) GetVelocityTrend() analytics.VelocityTrend {
	return s.projection.GetVelocityTrend()
}

// GetVelocityStats returns statistical summary of velocity.
func (s *ForecastService) GetVelocityStats() analytics.VelocityStats {
	return s.projection.GetVelocityStats()
}

// GetBurndown returns burndown chart data.
func (s *ForecastService) GetBurndown() ([]analytics.BurndownPoint, error) {
	forecast, err := s.GetForecast()
	if err != nil {
		return nil, err
	}
	if forecast == nil {
		return nil, nil
	}
	return forecast.Burndown, nil
}

// GetVelocityWindows returns velocity for each configured time window.
func (s *ForecastService) GetVelocityWindows() []analytics.VelocityWindow {
	return s.projection.GetVelocityWindows()
}

// SimpleForecast provides basic forecasting without the full service infrastructure.
// Useful for CLI commands that don't need all the complexity.
type SimpleForecast struct {
	Velocity       float64
	RemainingTasks int
	TotalTasks     int
	EstimatedDays  float64
	Trend          analytics.TrendDirection
	TrendSlope     float64
}

// GetSimpleForecast returns a simplified forecast result.
func (s *ForecastService) GetSimpleForecast() (*SimpleForecast, error) {
	forecast, err := s.GetForecast()
	if err != nil {
		return nil, err
	}
	if forecast == nil {
		return nil, nil
	}

	return &SimpleForecast{
		Velocity:       forecast.Velocity,
		RemainingTasks: forecast.RemainingTasks,
		TotalTasks:     forecast.TotalTasks,
		EstimatedDays:  forecast.EstimatedDays,
		Trend:          forecast.Trend.Direction,
		TrendSlope:     forecast.Trend.Slope,
	}, nil
}
