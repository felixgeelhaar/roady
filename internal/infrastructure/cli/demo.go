package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/spf13/cobra"
)

const (
	demoDefaultDir = "roady-demo"
)

var demoCmd = &cobra.Command{
	Use:   "demo [directory]",
	Short: "Scaffold a pre-seeded project with intentional drift to showcase Roady",
	Long: `demo creates a sample Roady project with a deliberately drifted spec/plan/state
trio so a new user sees Roady's core value (drift detection) in under a minute.

The directory defaults to ./` + demoDefaultDir + ` and must not already contain a .roady/ folder.
After scaffolding, demo runs drift detect and prints suggested next steps.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDemo,
}

func init() {
	demoCmd.Flags().Bool("force", false, "Recreate the demo directory if it already exists")
	RootCmd.AddCommand(demoCmd)
}

func runDemo(cmd *cobra.Command, args []string) error {
	target := demoDefaultDir
	if len(args) > 0 && args[0] != "" {
		target = args[0]
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve demo path: %w", err)
	}

	force, _ := cmd.Flags().GetBool("force")
	if err := prepareDemoDir(abs, force); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Scaffolding Roady demo in %s\n\n", abs)

	ws := wiring.NewWorkspace(abs)
	if err := ws.Repo.Initialize(); err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}

	if err := seedDemoArtifacts(ws); err != nil {
		return fmt.Errorf("seed demo artifacts: %w", err)
	}

	if err := ws.Audit.Log("demo.scaffolded", "cli", map[string]interface{}{
		"path": abs,
	}); err != nil {
		// Audit failure is informational here — the demo is meant to be
		// disposable. Surface but don't abort.
		fmt.Fprintf(out, "(warning) audit log failed: %v\n", err)
	}

	// BuildAppServices returns a non-nil services value alongside a warning
	// error when the AI provider isn't configured. Drift detection does not
	// need AI, so we tolerate that warning the same way `loadServices` does.
	services, buildErr := wiring.BuildAppServices(abs)
	if services == nil {
		return fmt.Errorf("build services for demo: %w", buildErr)
	}

	report, err := services.Drift.DetectDrift(cmd.Context())
	if err != nil {
		return fmt.Errorf("detect drift in demo: %w", err)
	}

	fmt.Fprintln(out, "Running `roady drift detect`...")
	fmt.Fprintln(out)
	if len(report.Issues) == 0 {
		fmt.Fprintln(out, "  (no drift detected — demo seed may need refresh)")
	} else {
		fmt.Fprintf(out, "Detected %d drift issues:\n", len(report.Issues))
		for _, issue := range report.Issues {
			fmt.Fprintf(out, "  - [%s] (%s/%s) %s\n",
				issue.Severity, issue.Type, issue.Category, issue.Message)
			if issue.Hint != "" {
				fmt.Fprintf(out, "      hint: %s\n", issue.Hint)
			}
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "What happened:")
	fmt.Fprintln(out, "  1. A spec was generated with two features.")
	fmt.Fprintln(out, "  2. A spec.lock.json snapshot was saved (intent baseline).")
	fmt.Fprintln(out, "  3. The spec was edited to add a third feature — without re-locking.")
	fmt.Fprintln(out, "  4. Roady detected the drift between intent and current spec.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Try next:")
	fmt.Fprintf(out, "  cd %s\n", target)
	fmt.Fprintln(out, "  roady drift detect       # see the same report")
	fmt.Fprintln(out, "  roady drift accept       # accept the drift, re-lock the spec")
	fmt.Fprintln(out, "  roady status             # full project view")
	fmt.Fprintln(out, "  roady setup claude-code  # connect an AI agent")
	fmt.Fprintln(out)
	return nil
}

func prepareDemoDir(abs string, force bool) error {
	roadyDir := filepath.Join(abs, ".roady")
	if _, err := os.Stat(roadyDir); err == nil {
		if !force {
			return fmt.Errorf("demo directory already initialised (.roady exists at %s); pass --force to recreate", abs)
		}
		if err := os.RemoveAll(roadyDir); err != nil {
			return fmt.Errorf("remove existing .roady: %w", err)
		}
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return fmt.Errorf("create demo dir: %w", err)
	}
	return nil
}

// seedDemoArtifacts writes a spec, an older spec.lock that intentionally
// diverges from the spec (one feature missing), an approved plan, and an
// empty execution state. The divergence is what surfaces as drift.
func seedDemoArtifacts(ws *wiring.Workspace) error {
	now := time.Now().UTC()

	baselineSpec := &spec.ProductSpec{
		ID:          "todo-app",
		Title:       "TODO App",
		Description: "Sample project illustrating Roady's spec/plan/drift loop.",
		Version:     "0.1.0",
		Features: []spec.Feature{
			{
				ID:          "tasks-crud",
				Title:       "Task CRUD",
				Description: "Create, read, update, and delete tasks.",
				Requirements: []spec.Requirement{
					{ID: "tasks-create", Title: "Create task", Description: "Persist a new task with a title.", Priority: "high"},
					{ID: "tasks-list", Title: "List tasks", Description: "Return all tasks in creation order.", Priority: "high"},
				},
			},
			{
				ID:          "auth",
				Title:       "Authentication",
				Description: "Email + password authentication.",
				Requirements: []spec.Requirement{
					{ID: "auth-signup", Title: "Sign up", Description: "Register a new user account.", Priority: "medium"},
				},
			},
		},
	}

	currentSpec := *baselineSpec
	currentSpec.Features = append(currentSpec.Features, spec.Feature{
		ID:          "notifications",
		Title:       "Email Notifications",
		Description: "Send a daily digest of pending tasks (added after lock — drifts).",
		Requirements: []spec.Requirement{
			{ID: "notif-digest", Title: "Daily digest", Description: "Email summary at 08:00 local.", Priority: "medium"},
		},
	})

	if err := ws.Repo.SaveSpec(&currentSpec); err != nil {
		return err
	}
	if err := ws.Repo.SaveSpecLock(baselineSpec); err != nil {
		return err
	}

	plan := &planning.Plan{
		ID:             "plan-demo-001",
		SpecID:         currentSpec.ID,
		ApprovalStatus: planning.ApprovalApproved,
		CreatedAt:      now,
		UpdatedAt:      now,
		Tasks: []planning.Task{
			{ID: "task-tasks-create", Title: "Implement create-task endpoint",
				Priority: planning.PriorityHigh, FeatureID: "tasks-crud"},
			{ID: "task-tasks-list", Title: "Implement list-tasks endpoint",
				Priority: planning.PriorityHigh, FeatureID: "tasks-crud",
				DependsOn: []string{"task-tasks-create"}},
			{ID: "task-auth-signup", Title: "Implement signup flow",
				Priority: planning.PriorityMedium, FeatureID: "auth"},
		},
	}
	if err := ws.Repo.SavePlan(plan); err != nil {
		return err
	}

	state := planning.NewExecutionState(currentSpec.ID)
	return ws.Repo.SaveState(state)
}
