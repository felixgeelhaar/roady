package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type Server struct {
	mcpServer *mcp.Server
	initSvc   *application.InitService
	specSvc   *application.SpecService
	planSvc   *application.PlanService
	driftSvc  *application.DriftService
	policySvc *application.PolicyService
	taskSvc   *application.TaskService
	aiSvc     *application.AIPlanningService
	gitSvc    *application.GitService
	syncSvc   *application.SyncService
	auditSvc  *application.AuditService
	depSvc    *application.DependencyService
	debtSvc   *application.DebtService
}

var (
	Version     = "dev"
	BuildCommit = "unknown"
	BuildDate   = "unknown"
)

func NewServer(root string) (*Server, error) {
	services, err := wiring.BuildAppServices(root)
	if err != nil {
		return nil, fmt.Errorf("build services: %w", err)
	}
	if services == nil {
		return nil, fmt.Errorf("services initialization returned nil")
	}

	info := mcp.ServerInfo{
		Name:    "roady",
		Version: Version,
	}

	s := &Server{
		mcpServer: mcp.NewServer(info,
			mcp.WithTitle("Roady MCP Server"),
			mcp.WithDescription("Roady exposes deterministic project state, plans, and drift analysis to MCP clients."),
			mcp.WithWebsiteURL("https://github.com/felixgeelhaar/roady"),
			mcp.WithBuildInfo(BuildCommit, BuildDate),
			mcp.WithInstructions("Use tools to read spec/plan, generate plans, detect drift, and transition tasks."),
		),
		initSvc:   services.Init,
		specSvc:   services.Spec,
		planSvc:   services.Plan,
		driftSvc:  services.Drift,
		policySvc: services.Policy,
		taskSvc:   services.Task,
		aiSvc:     services.AI,
		gitSvc:    services.Git,
		syncSvc:   services.Sync,
		auditSvc:  services.Audit,
		depSvc:    services.Dependency,
		debtSvc:   services.Debt,
	}

	s.registerTools()
	return s, nil
}

type InitArgs struct {
	Name string `json:"name" jsonschema:"description=The name of the project"`
}

type UpdatePlanArgs struct {
	Tasks []planning.Task `json:"tasks" jsonschema:"description=The list of tasks to define the plan"`
}

type SyncArgs struct {
	PluginPath string `json:"plugin_path" jsonschema:"description=Path to the syncer plugin binary"`
}

func (s *Server) registerTools() {
	// Tool: roady_init
	s.mcpServer.Tool("roady_init").
		Description("Initialize a new roady project in the current directory").
		Handler(s.handleInit)

	// Tool: roady_get_spec
	s.mcpServer.Tool("roady_get_spec").
		Description("Retrieve the current product specification").
		Handler(s.handleGetSpec)

	// Tool: roady_get_plan
	s.mcpServer.Tool("roady_get_plan").
		Description("Retrieve the current execution plan").
		Handler(s.handleGetPlan)

	// Tool: roady_get_state
	s.mcpServer.Tool("roady_get_state").
		Description("Retrieve the current execution state (task statuses)").
		Handler(s.handleGetState)

	// Tool: roady_generate_plan (Heuristic)
	s.mcpServer.Tool("roady_generate_plan").
		Description("Generate a basic plan from the spec using 1:1 heuristic (resets custom tasks unless they match features)").
		Handler(s.handleGeneratePlan)

	// Tool: roady_update_plan (Smart Injection)
	s.mcpServer.Tool("roady_update_plan").
		Description("Update the plan with a specific list of tasks (Smart Injection). Use this to propose complex architectures.").
		Handler(s.handleUpdatePlan)

	// Tool: roady_detect_drift
	s.mcpServer.Tool("roady_detect_drift").
		Description("Detect discrepancies between the current Spec and Plan").
		Handler(s.handleDetectDrift)

	// Tool: roady_accept_drift
	s.mcpServer.Tool("roady_accept_drift").
		Description("Accept the current drift by locking the spec snapshot").
		Handler(s.handleAcceptDrift)

	// Tool: roady_status
	s.mcpServer.Tool("roady_status").
		Description("Get a high-level summary of the project status").
		Handler(s.handleStatus)

	// Tool: roady_check_policy
	s.mcpServer.Tool("roady_check_policy").
		Description("Check if the current plan complies with execution policies (e.g., WIP limits)").
		Handler(s.handleCheckPolicy)

	// Tool: roady_transition_task
	s.mcpServer.Tool("roady_transition_task").
		Description("Transition a task to a new state (e.g., start, complete, block, stop)").
		Handler(s.handleTransitionTask)

	// Tool: roady_explain_spec
	s.mcpServer.Tool("roady_explain_spec").
		Description("Provide an AI-generated architectural walkthrough of the current specification").
		Handler(s.handleExplainSpec)

	// Tool: roady_approve_plan
	s.mcpServer.Tool("roady_approve_plan").
		Description("Approve the current plan for execution").
		Handler(s.handleApprovePlan)

	// Tool: roady_get_usage
	s.mcpServer.Tool("roady_get_usage").
		Description("Retrieve project usage and telemetry statistics").
		Handler(s.handleGetUsage)

	// Tool: roady_explain_drift
	s.mcpServer.Tool("roady_explain_drift").
		Description("Provide an AI-generated explanation and resolution steps for current project drift").
		Handler(s.handleExplainDrift)

	// Tool: roady_add_feature
	s.mcpServer.Tool("roady_add_feature").
		Description("Add a new feature to the product specification and sync to docs/backlog.md").
		Handler(s.handleAddFeature)

	// Tool: roady_forecast (Horizon 5)
	s.mcpServer.Tool("roady_forecast").
		Description("Predict project completion based on current task velocity").
		Handler(s.handleForecast)

	// Tool: roady_org_status (Horizon 4)
	s.mcpServer.Tool("roady_org_status").
		Description("Get a status overview of all Roady projects in the directory tree").
		Handler(s.handleOrgStatus)

	// Tool: roady_git_sync (Horizon 5)
	s.mcpServer.Tool("roady_git_sync").
		Description("Synchronize task statuses by scanning git commit messages for markers").
		Handler(s.handleGitSync)

	// Tool: roady_sync (External Plugins)
	s.mcpServer.Tool("roady_sync").
		Description("Sync the plan with an external system via a plugin binary").
		Handler(s.handleSync)

	// Tool: roady_deps_list (Horizon 5)
	s.mcpServer.Tool("roady_deps_list").
		Description("List all cross-repository dependencies").
		Handler(s.handleDepsList)

	// Tool: roady_deps_scan (Horizon 5)
	s.mcpServer.Tool("roady_deps_scan").
		Description("Scan health status of all dependent repositories").
		Handler(s.handleDepsScan)

	// Tool: roady_deps_graph (Horizon 5)
	s.mcpServer.Tool("roady_deps_graph").
		Description("Get dependency graph summary with optional cycle detection").
		Handler(s.handleDepsGraph)

	// Tool: roady_debt_report (Horizon 5)
	s.mcpServer.Tool("roady_debt_report").
		Description("Generate comprehensive debt report with category breakdown and top debtors").
		Handler(s.handleDebtReport)

	// Tool: roady_debt_summary (Horizon 5)
	s.mcpServer.Tool("roady_debt_summary").
		Description("Quick overview of debt status including health level and top debtor").
		Handler(s.handleDebtSummary)

	// Tool: roady_sticky_drift (Horizon 5)
	s.mcpServer.Tool("roady_sticky_drift").
		Description("Get sticky debt items (unresolved drift for more than 7 days)").
		Handler(s.handleStickyDrift)

	// Tool: roady_debt_trend (Horizon 5)
	s.mcpServer.Tool("roady_debt_trend").
		Description("Analyze drift trend over time").
		Handler(s.handleDebtTrend)
}

func (s *Server) handleForecast(ctx context.Context, args struct{}) (string, error) {
	velocity, err := s.auditSvc.GetVelocity()
	if err != nil {
		return "", fmt.Errorf("get velocity: %w", err)
	}

	plan, err := s.planSvc.GetPlan()
	if err != nil {
		return "", fmt.Errorf("load plan: %w", err)
	}
	state, err := s.planSvc.GetState()
	if err != nil {
		return "", fmt.Errorf("load state: %w", err)
	}
	remaining := 0
	for _, t := range plan.Tasks {
		if state.TaskStates[t.ID].Status != planning.StatusVerified {
			remaining++
		}
	}

	return fmt.Sprintf("Velocity: %.2f tasks/day. Remaining: %d. Estimated: %v days.", velocity, remaining, remaining), nil
}

func (s *Server) handleOrgStatus(ctx context.Context, args struct{}) (any, error) {
	// Simple string return for this prototype
	return "Organizational status check initiated. Use CLI for full table view.", nil
}

func (s *Server) handleGitSync(ctx context.Context, args struct{}) (any, error) {
	results, err := s.gitSvc.SyncMarkers(10)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Server) handleSync(ctx context.Context, args SyncArgs) (any, error) {
	results, err := s.syncSvc.SyncWithPlugin(args.PluginPath)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Server) handleExplainDrift(ctx context.Context, args struct{}) (string, error) {
	report, err := s.driftSvc.DetectDrift(ctx)
	if err != nil {
		return "", err
	}
	return s.aiSvc.ExplainDrift(ctx, report)
}

func (s *Server) handleAcceptDrift(ctx context.Context, args struct{}) (string, error) {
	if err := s.driftSvc.AcceptDrift(); err != nil {
		return "", err
	}
	return "Drift accepted and spec snapshot locked.", nil
}

type AddFeatureArgs struct {
	Title       string `json:"title" jsonschema:"description=The title of the new feature"`
	Description string `json:"description" jsonschema:"description=A detailed description of the feature"`
}

func (s *Server) handleAddFeature(ctx context.Context, args AddFeatureArgs) (string, error) {
	spec, err := s.specSvc.AddFeature(args.Title, args.Description)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully added feature '%s'. Intent synced to docs/backlog.md. Total features: %d", args.Title, len(spec.Features)), nil
}

func (s *Server) handleGetUsage(ctx context.Context, args struct{}) (any, error) {
	return s.planSvc.GetUsage()
}

func (s *Server) handleApprovePlan(ctx context.Context, args struct{}) (string, error) {
	err := s.planSvc.ApprovePlan()
	if err != nil {
		return "", err
	}
	return "Plan approved successfully", nil
}

func (s *Server) handleExplainSpec(ctx context.Context, args struct{}) (string, error) {
	return s.aiSvc.ExplainSpec(ctx)
}

type TransitionTaskArgs struct {
	TaskID   string `json:"task_id" jsonschema:"description=The ID of the task to transition"`
	Event    string `json:"event" jsonschema:"description=The transition event (start, complete, block, stop, unblock, reopen)"`
	Evidence string `json:"evidence,omitempty" jsonschema:"description=Optional evidence for the transition (e.g. commit hash)"`
}

// StatusArgs defines filter parameters for roady_status tool
type StatusArgs struct {
	Status   string `json:"status,omitempty" jsonschema:"description=Filter by status (comma-separated: pending,blocked,in_progress,done,verified)"`
	Priority string `json:"priority,omitempty" jsonschema:"description=Filter by priority (comma-separated: high,medium,low)"`
	Ready    bool   `json:"ready,omitempty" jsonschema:"description=Show only tasks ready to start (unlocked + pending)"`
	Blocked  bool   `json:"blocked,omitempty" jsonschema:"description=Show only blocked tasks"`
	Active   bool   `json:"active,omitempty" jsonschema:"description=Show only in-progress tasks"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=Limit number of tasks returned"`
	JSON     bool   `json:"json,omitempty" jsonschema:"description=Return structured JSON output instead of text"`
}

func (s *Server) handleTransitionTask(ctx context.Context, args TransitionTaskArgs) (string, error) {
	err := s.taskSvc.TransitionTask(args.TaskID, args.Event, "ai-agent", args.Evidence)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Task %s transitioned with event %s successfully", args.TaskID, args.Event), nil
}

func (s *Server) handleInit(ctx context.Context, args InitArgs) (string, error) {
	err := s.initSvc.InitializeProject(args.Name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Project %s initialized successfully", args.Name), nil
}

func (s *Server) handleGetSpec(ctx context.Context, args struct{}) (any, error) {
	spec, err := s.specSvc.GetSpec()
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func (s *Server) handleGetPlan(ctx context.Context, args struct{}) (any, error) {
	plan, err := s.planSvc.GetPlan()
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *Server) handleGetState(ctx context.Context, args struct{}) (any, error) {
	state, err := s.planSvc.GetState()
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (s *Server) handleGeneratePlan(ctx context.Context, args struct{}) (string, error) {
	plan, err := s.planSvc.GeneratePlan(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Plan generated with %d tasks. Plan ID: %s", len(plan.Tasks), plan.ID), nil
}

func (s *Server) handleUpdatePlan(ctx context.Context, args UpdatePlanArgs) (string, error) {
	plan, err := s.planSvc.UpdatePlan(args.Tasks)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Plan updated with %d tasks. Plan ID: %s", len(plan.Tasks), plan.ID), nil
}

func (s *Server) handleDetectDrift(ctx context.Context, args struct{}) (any, error) {
	report, err := s.driftSvc.DetectDrift(ctx)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func (s *Server) handleStatus(ctx context.Context, args StatusArgs) (any, error) {
	plan, err := s.planSvc.GetPlan()
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	state, err := s.planSvc.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Parse filters
	var statusFilters []planning.TaskStatus
	if args.Status != "" {
		for _, s := range strings.Split(args.Status, ",") {
			if parsed, err := planning.ParseTaskStatus(strings.TrimSpace(s)); err == nil {
				statusFilters = append(statusFilters, parsed)
			}
		}
	}

	var priorityFilters []planning.TaskPriority
	if args.Priority != "" {
		for _, p := range strings.Split(args.Priority, ",") {
			if parsed, err := planning.ParseTaskPriority(strings.TrimSpace(p)); err == nil {
				priorityFilters = append(priorityFilters, parsed)
			}
		}
	}

	// Filter tasks
	var filtered []planning.Task
	for _, t := range plan.Tasks {
		status := planning.StatusPending
		if res, ok := state.TaskStates[t.ID]; ok {
			status = res.Status
		}

		// Apply shortcut flags
		if args.Ready {
			if status != planning.StatusPending || !isTaskUnlockedByDeps(t, state) {
				continue
			}
		}
		if args.Blocked && status != planning.StatusBlocked {
			continue
		}
		if args.Active && status != planning.StatusInProgress {
			continue
		}

		// Apply status filter
		if len(statusFilters) > 0 && !containsStatus(statusFilters, status) {
			continue
		}

		// Apply priority filter
		if len(priorityFilters) > 0 && !containsPriority(priorityFilters, t.Priority) {
			continue
		}

		filtered = append(filtered, t)
	}

	// Apply limit
	if args.Limit > 0 && len(filtered) > args.Limit {
		filtered = filtered[:args.Limit]
	}

	// Count all tasks by status (for summary)
	counts := make(map[string]int)
	for _, t := range plan.Tasks {
		status := "pending"
		if res, ok := state.TaskStates[t.ID]; ok {
			status = string(res.Status)
		}
		counts[status]++
	}

	// JSON output
	if args.JSON {
		type taskOutput struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Status   string `json:"status"`
			Priority string `json:"priority"`
			Unlocked bool   `json:"unlocked,omitempty"`
		}

		tasks := make([]taskOutput, 0, len(filtered))
		for _, t := range filtered {
			status := planning.StatusPending
			if res, ok := state.TaskStates[t.ID]; ok {
				status = res.Status
			}
			tasks = append(tasks, taskOutput{
				ID:       t.ID,
				Title:    t.Title,
				Status:   string(status),
				Priority: string(t.Priority),
				Unlocked: status == planning.StatusPending && isTaskUnlockedByDeps(t, state),
			})
		}

		output := map[string]any{
			"total_tasks":    len(plan.Tasks),
			"filtered_count": len(filtered),
			"counts":         counts,
			"tasks":          tasks,
		}

		jsonBytes, err := json.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("marshal json: %w", err)
		}
		return string(jsonBytes), nil
	}

	// Text output
	statusStr := fmt.Sprintf("Tasks: %d total\n- Done: %d\n- In Progress: %d\n- Pending: %d\n- Blocked: %d",
		len(plan.Tasks), counts["done"]+counts["verified"], counts["in_progress"], counts["pending"], counts["blocked"])

	if len(statusFilters) > 0 || len(priorityFilters) > 0 || args.Ready || args.Blocked || args.Active {
		statusStr += fmt.Sprintf("\n\nFiltered Tasks: %d", len(filtered))
		for _, t := range filtered {
			status := planning.StatusPending
			if res, ok := state.TaskStates[t.ID]; ok {
				status = res.Status
			}
			statusStr += fmt.Sprintf("\n- [%s] %s (%s)", status, t.Title, t.Priority)
		}
	}

	return statusStr, nil
}

// isTaskUnlockedByDeps checks if all dependencies are complete
func isTaskUnlockedByDeps(task planning.Task, state *planning.ExecutionState) bool {
	for _, depID := range task.DependsOn {
		if state == nil {
			return false
		}
		if res, ok := state.TaskStates[depID]; ok {
			if !res.Status.IsComplete() {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// containsStatus checks if a status is in the list
func containsStatus(statuses []planning.TaskStatus, s planning.TaskStatus) bool {
	for _, status := range statuses {
		if status == s {
			return true
		}
	}
	return false
}

// containsPriority checks if a priority is in the list
func containsPriority(priorities []planning.TaskPriority, p planning.TaskPriority) bool {
	for _, priority := range priorities {
		if priority == p {
			return true
		}
	}
	return false
}

func (s *Server) handleCheckPolicy(ctx context.Context, args struct{}) (any, error) {
	vioations, err := s.policySvc.CheckCompliance()
	if err != nil {
		return nil, err
	}
	if len(vioations) == 0 {
		return "No policy violations found.", nil
	}
	return vioations, nil
}

func (s *Server) Start() error {
	return s.StartStdio()
}

func (s *Server) StartStdio() error {
	return s.ServeStdio(context.Background())
}

func (s *Server) StartHTTP(addr string) error {
	return s.ServeHTTP(context.Background(), addr)
}

func (s *Server) StartWebSocket(addr string) error {
	return s.ServeWebSocket(context.Background(), addr)
}

func (s *Server) ServeStdio(ctx context.Context) error {
	return mcp.ServeStdio(ctx, s.mcpServer)
}

func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	return mcp.ServeHTTP(ctx, s.mcpServer, addr, mcp.WithDefaultCORS())
}

func (s *Server) ServeWebSocket(ctx context.Context, addr string) error {
	return mcp.ServeWebSocket(ctx, s.mcpServer, addr)
}

func (s *Server) StartGRPC(addr string) error {
	return s.ServeGRPC(context.Background(), addr)
}

func (s *Server) ServeGRPC(ctx context.Context, addr string) error {
	return mcp.ServeGRPC(ctx, s.mcpServer, addr)
}

// Dependency MCP handlers

func (s *Server) handleDepsList(ctx context.Context, args struct{}) (any, error) {
	deps, err := s.depSvc.ListDependencies()
	if err != nil {
		return nil, fmt.Errorf("list dependencies: %w", err)
	}
	return deps, nil
}

func (s *Server) handleDepsScan(ctx context.Context, args struct{}) (any, error) {
	result, err := s.depSvc.ScanDependentRepos(nil)
	if err != nil {
		return nil, fmt.Errorf("scan dependencies: %w", err)
	}
	return result, nil
}

type DepsGraphArgs struct {
	CheckCycles bool `json:"check_cycles,omitempty" jsonschema:"description=Whether to check for cyclic dependencies"`
}

func (s *Server) handleDepsGraph(ctx context.Context, args DepsGraphArgs) (any, error) {
	summary, err := s.depSvc.GetDependencySummary()
	if err != nil {
		return nil, fmt.Errorf("get dependency summary: %w", err)
	}

	response := map[string]any{
		"summary": summary,
	}

	if args.CheckCycles {
		hasCycle, err := s.depSvc.CheckForCycles()
		if err != nil {
			return nil, fmt.Errorf("check cycles: %w", err)
		}
		response["has_cycle"] = hasCycle
	}

	return response, nil
}

// Debt MCP handlers

func (s *Server) handleDebtReport(ctx context.Context, args struct{}) (any, error) {
	report, err := s.debtSvc.GetDebtReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("get debt report: %w", err)
	}
	return report, nil
}

func (s *Server) handleDebtSummary(ctx context.Context, args struct{}) (any, error) {
	summary, err := s.debtSvc.GetDebtSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("get debt summary: %w", err)
	}
	return summary, nil
}

func (s *Server) handleStickyDrift(ctx context.Context, args struct{}) (any, error) {
	items, err := s.debtSvc.GetStickyDrift()
	if err != nil {
		return nil, fmt.Errorf("get sticky drift: %w", err)
	}
	return items, nil
}

type DebtTrendArgs struct {
	Days int `json:"days,omitempty" jsonschema:"description=Analysis window in days (default: 30)"`
}

func (s *Server) handleDebtTrend(ctx context.Context, args DebtTrendArgs) (any, error) {
	days := args.Days
	if days <= 0 {
		days = 30
	}
	trend, err := s.debtSvc.GetDriftTrend(days)
	if err != nil {
		return nil, fmt.Errorf("get debt trend: %w", err)
	}
	return trend, nil
}
