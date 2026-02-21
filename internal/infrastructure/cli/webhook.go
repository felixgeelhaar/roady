package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/infrastructure/webhook"
	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhook server for external integrations",
	Long: `The webhook command manages the HTTP server that receives events from external
project management tools like GitHub, Jira, and Linear.

Events are processed and used to update Roady task states in real-time.`,
}

func init() {
	RootCmd.AddCommand(webhookCmd)
	webhookCmd.AddCommand(webhookServeCmd)
}

var (
	webhookPort         int
	webhookGitHubSecret string
	webhookJiraSecret   string
	webhookLinearSecret string
)

var webhookServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the webhook HTTP server",
	Long: `Start the webhook HTTP server to receive events from external systems.

The server exposes endpoints for GitHub, Jira, and Linear webhooks:
  - POST /webhooks/github
  - POST /webhooks/jira
  - POST /webhooks/linear

Configure your external tools to send webhooks to these endpoints.

Secrets can be provided to validate webhook signatures:
  --github-secret: GitHub webhook secret
  --jira-secret: Jira webhook secret (query param or Bearer token)
  --linear-secret: Linear webhook signing secret`,
	Example: `  # Start server on port 8080
  roady webhook serve --port 8080

  # Start with GitHub webhook secret
  roady webhook serve --port 8080 --github-secret=your-secret

  # Start with all secrets from environment
  GITHUB_WEBHOOK_SECRET=xxx JIRA_WEBHOOK_SECRET=yyy LINEAR_WEBHOOK_SECRET=zzz roady webhook serve`,
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		// Get secrets from environment if not provided
		githubSecret := webhookGitHubSecret
		jiraSecret := webhookJiraSecret
		linearSecret := webhookLinearSecret

		if githubSecret == "" {
			githubSecret = os.Getenv("GITHUB_WEBHOOK_SECRET")
		}
		if jiraSecret == "" {
			jiraSecret = os.Getenv("JIRA_WEBHOOK_SECRET")
		}
		if linearSecret == "" {
			linearSecret = os.Getenv("LINEAR_WEBHOOK_SECRET")
		}

		// Create processor
		processor := newWebhookProcessor(services)

		// Create server
		addr := fmt.Sprintf(":%d", webhookPort)
		server := webhook.NewServer(addr, processor)

		// Register handlers
		server.RegisterHandler(webhook.NewGitHubHandler())
		server.RegisterHandler(webhook.NewJiraHandler())
		server.RegisterHandler(webhook.NewLinearHandler())

		// Set secrets
		if githubSecret != "" {
			server.SetSecret("github", githubSecret)
			fmt.Println("GitHub webhook signature validation enabled")
		}
		if jiraSecret != "" {
			server.SetSecret("jira", jiraSecret)
			fmt.Println("Jira webhook signature validation enabled")
		}
		if linearSecret != "" {
			server.SetSecret("linear", linearSecret)
			fmt.Println("Linear webhook signature validation enabled")
		}

		// Handle graceful shutdown
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-stop
			fmt.Println("\nShutting down webhook server...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		}()

		fmt.Printf("Starting webhook server on %s\n", addr)
		fmt.Println("Endpoints:")
		fmt.Printf("  POST http://localhost:%d/webhooks/github\n", webhookPort)
		fmt.Printf("  POST http://localhost:%d/webhooks/jira\n", webhookPort)
		fmt.Printf("  POST http://localhost:%d/webhooks/linear\n", webhookPort)
		fmt.Printf("  GET  http://localhost:%d/health\n", webhookPort)
		fmt.Printf("  GET  http://localhost:%d/events\n", webhookPort)
		fmt.Println("\nPress Ctrl+C to stop")

		if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	},
}

func init() {
	webhookServeCmd.Flags().IntVarP(&webhookPort, "port", "p", 8080, "Port to listen on")
	webhookServeCmd.Flags().StringVar(&webhookGitHubSecret, "github-secret", "", "GitHub webhook secret")
	webhookServeCmd.Flags().StringVar(&webhookJiraSecret, "jira-secret", "", "Jira webhook secret")
	webhookServeCmd.Flags().StringVar(&webhookLinearSecret, "linear-secret", "", "Linear webhook secret")
}

// webhookProcessor processes incoming webhook events.
type webhookProcessor struct {
	services *wiring.AppServices
}

func newWebhookProcessor(services *wiring.AppServices) *webhookProcessor {
	return &webhookProcessor{services: services}
}

func (p *webhookProcessor) ProcessEvent(ctx context.Context, event *webhook.Event) error {
	if event.TaskID == "" {
		// Event doesn't have a roady-id marker, skip processing
		return nil
	}

	// Load current state
	state, err := p.services.Plan.GetState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	if state == nil {
		state = planning.NewExecutionState("")
	}

	// Get current task status
	taskResult, ok := state.TaskStates[event.TaskID]
	if !ok {
		// Task not found, create entry
		taskResult = planning.TaskResult{
			Status:       planning.StatusPending,
			ExternalRefs: make(map[string]planning.ExternalRef),
		}
	}

	// Update status if changed
	statusChanged := false
	if event.Status != "" && event.Status != taskResult.Status {
		taskResult.Status = event.Status
		statusChanged = true
	}

	// Update external ref
	if taskResult.ExternalRefs == nil {
		taskResult.ExternalRefs = make(map[string]planning.ExternalRef)
	}
	taskResult.ExternalRefs[event.Provider] = planning.ExternalRef{
		ID:           event.ExternalID,
		LastSyncedAt: event.Timestamp,
	}

	// Store updated result
	state.TaskStates[event.TaskID] = taskResult

	// Save state via workspace repository
	if err := p.services.Workspace.Repo.SaveState(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	if statusChanged {
		fmt.Printf("Updated task %s status to %s (from %s webhook)\n", event.TaskID, event.Status, event.Provider)
	}

	return nil
}
