package wiring

import (
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/ai"
	"github.com/felixgeelhaar/roady/pkg/application"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// AppServices exposes the application layer services wired together with a workspace.
type AppServices struct {
	Workspace  *Workspace
	Init       *application.InitService
	Spec       *application.SpecService
	Plan       *application.PlanService
	Drift      *application.DriftService
	Policy     *application.PolicyService
	Task       *application.TaskService
	AI         *application.AIPlanningService
	Git        *application.GitService
	Sync       *application.SyncService
	Audit      *application.AuditService // Concrete service for read operations like GetVelocity
	Usage      *application.UsageService // Usage tracking service (separate from audit)
	Forecast   *application.ForecastService
	Dependency *application.DependencyService
	Debt       *application.DebtService // Debt analysis service (Horizon 5)
	Provider   domainai.Provider
}

// BuildAppServices constructs the workbench of services and AI provider wiring for a repo root.
func BuildAppServices(root string) (*AppServices, error) {
	workspace := NewWorkspace(root)
	provider, err := LoadAIProvider(root)
	var loadErr error
	if err != nil {
		loadErr = fmt.Errorf("AI provider config fallback: %w", err)
		fallback, fallbackErr := ai.GetDefaultProvider("ollama", "llama3")
		if fallbackErr != nil {
			return nil, fmt.Errorf("fallback AI provider failed: %w", fallbackErr)
		}
		provider = ai.NewResilientProvider(fallback)
	}

	// Create services in dependency order
	policySvc := application.NewPolicyService(workspace.Repo)
	planSvc := application.NewPlanService(workspace.Repo, workspace.Audit)
	taskSvc := application.NewTaskService(workspace.Repo, workspace.Audit, policySvc)
	driftSvc := application.NewDriftService(workspace.Repo, workspace.Audit, storage.NewCodebaseInspector(), policySvc)
	aiSvc := application.NewAIPlanningService(workspace.Repo, provider, workspace.Audit, planSvc)
	debtSvc := application.NewDebtService(driftSvc, workspace.Audit)

	// Create velocity projection for forecasting and hydrate from stored events
	velocityProjection := events.NewExtendedVelocityProjection(7, 14, 30)
	if storedEvents, err := workspace.Repo.LoadEvents(); err == nil {
		for _, ev := range storedEvents {
			baseEvent := &events.BaseEvent{
				Type:      ev.Action,
				Timestamp: ev.Timestamp,
				Metadata:  ev.Metadata,
			}
			_ = velocityProjection.Apply(baseEvent)
		}
	}
	forecastSvc := application.NewForecastService(velocityProjection, workspace.Repo)

	// Create dependency service
	depSvc := application.NewDependencyService(workspace.Repo, root)

	services := &AppServices{
		Workspace:  workspace,
		Init:       application.NewInitService(workspace.Repo, workspace.Audit),
		Spec:       application.NewSpecService(workspace.Repo),
		Plan:       planSvc,
		Drift:      driftSvc,
		Policy:     policySvc,
		Task:       taskSvc,
		AI:         aiSvc,
		Git:        application.NewGitService(workspace.Repo, taskSvc),
		Sync:       application.NewSyncServiceWithPlugins(workspace.Repo, workspace.Repo, taskSvc),
		Audit:      workspace.Audit,
		Usage:      workspace.Usage,
		Forecast:   forecastSvc,
		Dependency: depSvc,
		Debt:       debtSvc,
		Provider:   provider,
	}

	return services, loadErr
}

// BuildAppServicesWithProvider allows callers to supply a custom AI provider resolver.
func BuildAppServicesWithProvider(root string, resolver func(string) (domainai.Provider, error)) (*AppServices, error) {
	workspace := NewWorkspace(root)
	provider, err := resolver(root)
	var loadErr error
	if err != nil {
		loadErr = fmt.Errorf("AI provider config fallback: %w", err)
		fallback, fallbackErr := ai.GetDefaultProvider("ollama", "llama3")
		if fallbackErr != nil {
			return nil, fmt.Errorf("fallback AI provider failed: %w", fallbackErr)
		}
		provider = ai.NewResilientProvider(fallback)
	}

	// Create services in dependency order
	policySvc := application.NewPolicyService(workspace.Repo)
	planSvc := application.NewPlanService(workspace.Repo, workspace.Audit)
	taskSvc := application.NewTaskService(workspace.Repo, workspace.Audit, policySvc)
	driftSvc := application.NewDriftService(workspace.Repo, workspace.Audit, storage.NewCodebaseInspector(), policySvc)
	aiSvc := application.NewAIPlanningService(workspace.Repo, provider, workspace.Audit, planSvc)
	debtSvc := application.NewDebtService(driftSvc, workspace.Audit)

	// Create velocity projection for forecasting and hydrate from stored events
	velocityProjection := events.NewExtendedVelocityProjection(7, 14, 30)
	if storedEvents, err := workspace.Repo.LoadEvents(); err == nil {
		for _, ev := range storedEvents {
			baseEvent := &events.BaseEvent{
				Type:      ev.Action,
				Timestamp: ev.Timestamp,
				Metadata:  ev.Metadata,
			}
			_ = velocityProjection.Apply(baseEvent)
		}
	}
	forecastSvc := application.NewForecastService(velocityProjection, workspace.Repo)

	// Create dependency service
	depSvc := application.NewDependencyService(workspace.Repo, root)

	services := &AppServices{
		Workspace:  workspace,
		Init:       application.NewInitService(workspace.Repo, workspace.Audit),
		Spec:       application.NewSpecService(workspace.Repo),
		Plan:       planSvc,
		Drift:      driftSvc,
		Policy:     policySvc,
		Task:       taskSvc,
		AI:         aiSvc,
		Git:        application.NewGitService(workspace.Repo, taskSvc),
		Sync:       application.NewSyncServiceWithPlugins(workspace.Repo, workspace.Repo, taskSvc),
		Audit:      workspace.Audit,
		Usage:      workspace.Usage,
		Forecast:   forecastSvc,
		Dependency: depSvc,
		Debt:       debtSvc,
		Provider:   provider,
	}

	return services, loadErr
}
