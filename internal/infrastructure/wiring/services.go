package wiring

import (
	"context"
	"fmt"
	"path/filepath"

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
	Billing    *application.BillingService
	AI         *application.AIPlanningService
	Git        *application.GitService
	Sync       *application.SyncService
	Audit      *application.EventSourcedAuditService // Event-sourced audit with dispatcher and projections
	Usage      *application.UsageService             // Usage tracking service (separate from audit)
	Forecast   *application.ForecastService
	Dependency *application.DependencyService
	Debt       *application.DebtService // Debt analysis service (Horizon 5)
	Plugin     *application.PluginService
	Team       *application.TeamService
	Publisher  *storage.InMemoryEventPublisher
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

	return buildServicesWithProvider(workspace, root, provider, loadErr)
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

	return buildServicesWithProvider(workspace, root, provider, loadErr)
}

// buildServicesWithProvider is the shared implementation for building app services.
func buildServicesWithProvider(workspace *Workspace, root string, provider domainai.Provider, loadErr error) (*AppServices, error) {
	// Create event store and publisher for event-sourced audit
	eventStore, err := storage.NewFileEventStore(filepath.Join(root, storage.RoadyDir))
	if err != nil {
		return nil, fmt.Errorf("create event store: %w", err)
	}
	publisher := storage.NewInMemoryEventPublisher()

	// Create event-sourced audit service with dispatcher and projections
	auditSvc, err := application.NewEventSourcedAuditService(eventStore, publisher)
	if err != nil {
		return nil, fmt.Errorf("create event-sourced audit: %w", err)
	}

	// Create and wire event dispatcher with handlers
	dispatcher := events.NewEventDispatcher()
	dispatcher.Register(events.NewLoggingHandler(nil).Registration())
	dispatcher.Register(events.NewDriftWarningHandler(nil, nil).Registration())
	dispatcher.Register(events.NewTaskTransitionHandler(nil).Registration())
	auditSvc.SetDispatcher(dispatcher)

	// Create services in dependency order
	policySvc := application.NewPolicyService(workspace.Repo)
	planSvc := application.NewPlanService(workspace.Repo, auditSvc)
	taskSvc := application.NewTaskService(workspace.Repo, auditSvc, policySvc)
	driftSvc := application.NewDriftService(workspace.Repo, auditSvc, storage.NewCodebaseInspector(), policySvc)
	aiSvc := application.NewAIPlanningService(workspace.Repo, provider, auditSvc, planSvc)
	debtSvc := application.NewDebtService(driftSvc, auditSvc)

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

	// Subscribe velocity projection to live events via publisher
	publisher.Subscribe(func(e *events.BaseEvent) error {
		return velocityProjection.Apply(e)
	})

	// Subscribe webhook notifier to live events if configured
	if workspace.Notifier != nil {
		notifier := workspace.Notifier
		publisher.Subscribe(func(e *events.BaseEvent) error {
			notifier.Notify(context.Background(), e)
			return nil
		})
	}

	forecastSvc := application.NewForecastService(velocityProjection, workspace.Repo)

	// Create dependency service
	depSvc := application.NewDependencyService(workspace.Repo, root)

	services := &AppServices{
		Workspace:  workspace,
		Init:       application.NewInitService(workspace.Repo, auditSvc),
		Spec:       application.NewSpecService(workspace.Repo),
		Plan:       planSvc,
		Drift:      driftSvc,
		Policy:     policySvc,
		Task:       taskSvc,
		Billing:    application.NewBillingService(workspace.Repo),
		AI:         aiSvc,
		Git:        application.NewGitService(workspace.Repo, taskSvc),
		Sync:       application.NewSyncServiceWithPlugins(workspace.Repo, workspace.Repo, taskSvc),
		Audit:      auditSvc,
		Usage:      workspace.Usage,
		Forecast:   forecastSvc,
		Dependency: depSvc,
		Debt:       debtSvc,
		Plugin:     application.NewPluginService(workspace.Repo),
		Team:       application.NewTeamService(workspace.Repo, auditSvc),
		Publisher:  publisher,
		Provider:   provider,
	}

	return services, loadErr
}
