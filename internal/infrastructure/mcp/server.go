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
	"github.com/felixgeelhaar/roady/pkg/domain/team"
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
	auditSvc  *application.EventSourcedAuditService
	depSvc       *application.DependencyService
	debtSvc      *application.DebtService
	forecastSvc  *application.ForecastService
	orgSvc       *application.OrgService
	pluginSvc    *application.PluginService
	teamSvc      *application.TeamService
	root         string
}

var (
	Version     = "dev"
	BuildCommit = "unknown"
	BuildDate   = "unknown"
)

// mcpErr returns a user-friendly error for MCP clients.
// Internal details are omitted — only the friendly message is returned.
func mcpErr(friendly string) error {
	return fmt.Errorf("%s", friendly)
}

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
		depSvc:      services.Dependency,
		debtSvc:     services.Debt,
		forecastSvc: services.Forecast,
		orgSvc:      application.NewOrgService(root),
		pluginSvc:   application.NewPluginService(services.Workspace.Repo),
		teamSvc:     services.Team,
		root:        root,
	}

	s.registerTools()
	s.registerApps()
	s.registerSchemaResource()
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
		UIResource("ui://roady/init").
		Handler(s.handleInit)

	// Tool: roady_get_spec
	s.mcpServer.Tool("roady_get_spec").
		Description("Retrieve the current product specification").
		UIResource("ui://roady/spec").
		Handler(s.handleGetSpec)

	// Tool: roady_get_plan
	s.mcpServer.Tool("roady_get_plan").
		Description("Retrieve the current execution plan").
		UIResource("ui://roady/plan").
		Handler(s.handleGetPlan)

	// Tool: roady_get_state
	s.mcpServer.Tool("roady_get_state").
		Description("Retrieve the current execution state (task statuses)").
		UIResource("ui://roady/state").
		Handler(s.handleGetState)

	// Tool: roady_generate_plan (Heuristic)
	s.mcpServer.Tool("roady_generate_plan").
		Description("Generate a basic plan from the spec using 1:1 heuristic (resets custom tasks unless they match features)").
		UIResource("ui://roady/plan").
		Handler(s.handleGeneratePlan)

	// Tool: roady_update_plan (Smart Injection)
	s.mcpServer.Tool("roady_update_plan").
		Description("Update the plan with a specific list of tasks (Smart Injection). Use this to propose complex architectures.").
		UIResource("ui://roady/plan").
		Handler(s.handleUpdatePlan)

	// Tool: roady_detect_drift
	s.mcpServer.Tool("roady_detect_drift").
		Description("Detect discrepancies between the current Spec and Plan").
		UIResource("ui://roady/drift").
		Handler(s.handleDetectDrift)

	// Tool: roady_accept_drift
	s.mcpServer.Tool("roady_accept_drift").
		Description("Accept the current drift by locking the spec snapshot").
		UIResource("ui://roady/drift").
		Handler(s.handleAcceptDrift)

	// Tool: roady_status
	s.mcpServer.Tool("roady_status").
		Description("Get a high-level summary of the project status").
		UIResource("ui://roady/status").
		Handler(s.handleStatus)

	// Tool: roady_check_policy
	s.mcpServer.Tool("roady_check_policy").
		Description("Check if the current plan complies with execution policies (e.g., WIP limits)").
		UIResource("ui://roady/policy").
		Handler(s.handleCheckPolicy)

	// Tool: roady_transition_task
	s.mcpServer.Tool("roady_transition_task").
		Description("Transition a task to a new state (e.g., start, complete, block, stop)").
		UIResource("ui://roady/state").
		Handler(s.handleTransitionTask)

	// Tool: roady_explain_spec
	s.mcpServer.Tool("roady_explain_spec").
		Description("Provide an AI-generated architectural walkthrough of the current specification").
		UIResource("ui://roady/spec").
		Handler(s.handleExplainSpec)

	// Tool: roady_approve_plan
	s.mcpServer.Tool("roady_approve_plan").
		Description("Approve the current plan for execution").
		UIResource("ui://roady/plan").
		Handler(s.handleApprovePlan)

	// Tool: roady_get_usage
	s.mcpServer.Tool("roady_get_usage").
		Description("Retrieve project usage and telemetry statistics").
		UIResource("ui://roady/usage").
		Handler(s.handleGetUsage)

	// Tool: roady_explain_drift
	s.mcpServer.Tool("roady_explain_drift").
		Description("Provide an AI-generated explanation and resolution steps for current project drift").
		UIResource("ui://roady/drift").
		Handler(s.handleExplainDrift)

	// Tool: roady_add_feature
	s.mcpServer.Tool("roady_add_feature").
		Description("Add a new feature to the product specification and sync to docs/backlog.md").
		UIResource("ui://roady/spec").
		Handler(s.handleAddFeature)

	// Tool: roady_forecast (Horizon 5)
	s.mcpServer.Tool("roady_forecast").
		Description("Predict project completion based on current task velocity").
		UIResource("ui://roady/forecast").
		Handler(s.handleForecast)

	// Tool: roady_org_status (Horizon 4)
	s.mcpServer.Tool("roady_org_status").
		Description("Get a status overview of all Roady projects in the directory tree").
		UIResource("ui://roady/org").
		Handler(s.handleOrgStatus)

	// Tool: roady_git_sync (Horizon 5)
	s.mcpServer.Tool("roady_git_sync").
		Description("Synchronize task statuses by scanning git commit messages for markers").
		UIResource("ui://roady/git-sync").
		Handler(s.handleGitSync)

	// Tool: roady_sync (External Plugins)
	s.mcpServer.Tool("roady_sync").
		Description("Sync the plan with an external system via a plugin binary").
		UIResource("ui://roady/sync").
		Handler(s.handleSync)

	// Tool: roady_deps_list (Horizon 5)
	s.mcpServer.Tool("roady_deps_list").
		Description("List all cross-repository dependencies").
		UIResource("ui://roady/deps").
		Handler(s.handleDepsList)

	// Tool: roady_deps_scan (Horizon 5)
	s.mcpServer.Tool("roady_deps_scan").
		Description("Scan health status of all dependent repositories").
		UIResource("ui://roady/deps").
		Handler(s.handleDepsScan)

	// Tool: roady_deps_graph (Horizon 5)
	s.mcpServer.Tool("roady_deps_graph").
		Description("Get dependency graph summary with optional cycle detection").
		UIResource("ui://roady/deps").
		Handler(s.handleDepsGraph)

	// Tool: roady_debt_report (Horizon 5)
	s.mcpServer.Tool("roady_debt_report").
		Description("Generate comprehensive debt report with category breakdown and top debtors").
		UIResource("ui://roady/debt").
		Handler(s.handleDebtReport)

	// Tool: roady_debt_summary (Horizon 5)
	s.mcpServer.Tool("roady_debt_summary").
		Description("Quick overview of debt status including health level and top debtor").
		UIResource("ui://roady/debt").
		Handler(s.handleDebtSummary)

	// Tool: roady_sticky_drift (Horizon 5)
	s.mcpServer.Tool("roady_sticky_drift").
		Description("Get sticky debt items (unresolved drift for more than 7 days)").
		UIResource("ui://roady/debt").
		Handler(s.handleStickyDrift)

	// Tool: roady_debt_trend (Horizon 5)
	s.mcpServer.Tool("roady_debt_trend").
		Description("Analyze drift trend over time").
		UIResource("ui://roady/debt").
		Handler(s.handleDebtTrend)

	// Tool: roady_org_policy (v0.7.0)
	s.mcpServer.Tool("roady_org_policy").
		Description("Get merged policy for a project (org defaults + project overrides)").
		UIResource("ui://roady/org").
		Handler(s.handleOrgPolicy)

	// Tool: roady_org_detect_drift (v0.7.0)
	s.mcpServer.Tool("roady_org_detect_drift").
		Description("Detect drift across all projects in the directory tree").
		UIResource("ui://roady/org").
		Handler(s.handleOrgDetectDrift)

	// Tool: roady_plugin_list (v0.7.0)
	s.mcpServer.Tool("roady_plugin_list").
		Description("List all registered plugins with their status").
		UIResource("ui://roady/plugins").
		Handler(s.handlePluginList)

	// Tool: roady_plugin_validate (v0.7.0)
	s.mcpServer.Tool("roady_plugin_validate").
		Description("Validate a registered plugin by loading and initializing it").
		UIResource("ui://roady/plugins").
		Handler(s.handlePluginValidate)

	// Tool: roady_plugin_status (v0.7.0)
	s.mcpServer.Tool("roady_plugin_status").
		Description("Check health status of one or all plugins").
		UIResource("ui://roady/plugins").
		Handler(s.handlePluginStatus)

	// Tool: roady_messaging_list (v0.7.0)
	s.mcpServer.Tool("roady_messaging_list").
		Description("List configured messaging adapters").
		UIResource("ui://roady/messaging").
		Handler(s.handleMessagingList)

	// Tool: roady_query (v0.8.0)
	s.mcpServer.Tool("roady_query").
		Description("Ask a natural language question about the project and get an AI-generated answer").
		UIResource("ui://roady/status").
		Handler(s.handleQuery)

	// Tool: roady_suggest_priorities (v0.8.0)
	s.mcpServer.Tool("roady_suggest_priorities").
		Description("AI-powered priority suggestions based on spec analysis and task dependencies").
		UIResource("ui://roady/plan").
		Handler(s.handleSuggestPriorities)

	// Tool: roady_review_spec (v0.8.0)
	s.mcpServer.Tool("roady_review_spec").
		Description("Perform an AI-powered quality review of the current specification, returning a score and structured findings").
		UIResource("ui://roady/spec").
		Handler(s.handleReviewSpec)

	// Tool: roady_assign_task (v0.8.0)
	s.mcpServer.Tool("roady_assign_task").
		Description("Assign a task to a person or agent without changing its status").
		UIResource("ui://roady/state").
		Handler(s.handleAssignTask)

	// Tool: roady_get_snapshot (v0.6.0 - Coordinator)
	s.mcpServer.Tool("roady_get_snapshot").
		Description("Get a consistent project snapshot with progress, categorized task counts, and task lists").
		UIResource("ui://roady/status").
		Handler(s.handleGetSnapshot)

	// Tool: roady_get_ready_tasks (v0.6.0 - Coordinator)
	s.mcpServer.Tool("roady_get_ready_tasks").
		Description("Get tasks that are ready to start (unlocked and pending)").
		UIResource("ui://roady/status").
		Handler(s.handleGetReadyTasks)

	// Tool: roady_get_blocked_tasks (v0.6.0 - Coordinator)
	s.mcpServer.Tool("roady_get_blocked_tasks").
		Description("Get tasks that are currently blocked").
		UIResource("ui://roady/status").
		Handler(s.handleGetBlockedTasks)

	// Tool: roady_get_in_progress_tasks (v0.6.0 - Coordinator)
	s.mcpServer.Tool("roady_get_in_progress_tasks").
		Description("Get tasks that are currently in progress").
		UIResource("ui://roady/status").
		Handler(s.handleGetInProgressTasks)

	// Tool: roady_workspace_push (v0.8.0)
	s.mcpServer.Tool("roady_workspace_push").
		Description("Commit and push .roady/ workspace state to git remote").
		UIResource("ui://roady/workspace").
		Handler(s.handleWorkspacePush)

	// Tool: roady_workspace_pull (v0.8.0)
	s.mcpServer.Tool("roady_workspace_pull").
		Description("Pull remote .roady/ workspace changes and merge with conflict detection").
		UIResource("ui://roady/workspace").
		Handler(s.handleWorkspacePull)

	// Tool: roady_smart_decompose (v0.8.0)
	s.mcpServer.Tool("roady_smart_decompose").
		Description("AI-powered context-aware task decomposition using codebase structure analysis").
		UIResource("ui://roady/plan").
		Handler(s.handleSmartDecompose)

	// Tool: roady_team_list (v0.8.0)
	s.mcpServer.Tool("roady_team_list").
		Description("List all team members and their roles").
		UIResource("ui://roady/team").
		Handler(s.handleTeamList)

	// Tool: roady_team_add (v0.8.0)
	s.mcpServer.Tool("roady_team_add").
		Description("Add or update a team member with a role (admin, member, viewer)").
		UIResource("ui://roady/team").
		Handler(s.handleTeamAdd)

	// Tool: roady_team_remove (v0.8.0)
	s.mcpServer.Tool("roady_team_remove").
		Description("Remove a team member").
		UIResource("ui://roady/team").
		Handler(s.handleTeamRemove)
}

func (s *Server) handleForecast(ctx context.Context, args struct{}) (any, error) {
	forecast, err := s.forecastSvc.GetForecast()
	if err != nil {
		return nil, mcpErr("Unable to generate forecast. Ensure a plan exists and tasks have been transitioned.")
	}
	if forecast == nil {
		return "No plan found. Generate a plan first.", nil
	}

	// Build a JSON-friendly response with burndown data for the UI
	type burndownPt struct {
		Date      string `json:"date"`
		Actual    int    `json:"actual"`
		Projected int    `json:"projected"`
	}
	type windowPt struct {
		Days     int     `json:"days"`
		Velocity float64 `json:"velocity"`
		Count    int     `json:"count"`
	}
	type forecastResp struct {
		Remaining      int          `json:"remaining"`
		Completed      int          `json:"completed"`
		Total          int          `json:"total"`
		Velocity       float64      `json:"velocity"`
		EstimatedDays  float64      `json:"estimated_days"`
		CompletionRate float64      `json:"completion_rate"`
		Trend          string       `json:"trend"`
		TrendSlope     float64      `json:"trend_slope"`
		Confidence     float64      `json:"confidence"`
		CILow          float64      `json:"ci_low"`
		CIExpected     float64      `json:"ci_expected"`
		CIHigh         float64      `json:"ci_high"`
		Burndown       []burndownPt `json:"burndown"`
		Windows        []windowPt   `json:"windows"`
		DataPoints     int          `json:"data_points"`
	}

	resp := forecastResp{
		Remaining:      forecast.RemainingTasks,
		Completed:      forecast.CompletedTasks,
		Total:          forecast.TotalTasks,
		Velocity:       forecast.Velocity,
		EstimatedDays:  forecast.EstimatedDays,
		CompletionRate: forecast.CompletionRate(),
		Trend:          string(forecast.Trend.Direction),
		TrendSlope:     forecast.Trend.Slope,
		Confidence:     forecast.Trend.Confidence,
		CILow:          forecast.ConfidenceInterval.Low,
		CIExpected:     forecast.ConfidenceInterval.Expected,
		CIHigh:         forecast.ConfidenceInterval.High,
		DataPoints:     forecast.DataPoints,
	}

	for _, bp := range forecast.Burndown {
		resp.Burndown = append(resp.Burndown, burndownPt{
			Date:      bp.Date.Format("2006-01-02"),
			Actual:    bp.Actual,
			Projected: bp.Projected,
		})
	}

	for _, w := range forecast.Trend.Windows {
		resp.Windows = append(resp.Windows, windowPt{
			Days:     w.Days,
			Velocity: w.Velocity,
			Count:    w.Count,
		})
	}

	return resp, nil
}

func (s *Server) handleOrgStatus(ctx context.Context, args struct{}) (any, error) {
	orgSvc := s.orgSvc
	if orgSvc == nil {
		return "Org service not available.", nil
	}
	metrics, err := orgSvc.AggregateMetrics()
	if err != nil {
		return nil, mcpErr("Failed to aggregate org metrics.")
	}
	return metrics, nil
}

func (s *Server) handleGitSync(ctx context.Context, args struct{}) (any, error) {
	results, err := s.gitSvc.SyncMarkers(10)
	if err != nil {
		return nil, mcpErr("Failed to sync git markers. Ensure you are in a git repository with commit history.")
	}
	return results, nil
}

func (s *Server) handleSync(ctx context.Context, args SyncArgs) (any, error) {
	results, err := s.syncSvc.SyncWithPlugin(args.PluginPath)
	if err != nil {
		return nil, mcpErr("Failed to sync with plugin. Ensure the plugin binary exists and is executable.")
	}
	return results, nil
}

func (s *Server) handleExplainDrift(ctx context.Context, args struct{}) (string, error) {
	report, err := s.driftSvc.DetectDrift(ctx)
	if err != nil {
		return "", mcpErr("Failed to detect drift. Ensure both spec and plan exist.")
	}
	result, err := s.aiSvc.ExplainDrift(ctx, report)
	if err != nil {
		return "", mcpErr("Failed to generate drift explanation. Check your AI provider configuration.")
	}
	return result, nil
}

func (s *Server) handleAcceptDrift(ctx context.Context, args struct{}) (string, error) {
	if err := s.driftSvc.AcceptDrift(); err != nil {
		return "", mcpErr("Failed to accept drift. Ensure a spec exists.")
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
		return "", mcpErr("Failed to add feature. Ensure the project is initialized with a valid spec.")
	}
	return fmt.Sprintf("Successfully added feature '%s'. Intent synced to docs/backlog.md. Total features: %d", args.Title, len(spec.Features)), nil
}

func (s *Server) handleGetUsage(ctx context.Context, args struct{}) (any, error) {
	usage, err := s.planSvc.GetUsage()
	if err != nil {
		return nil, mcpErr("Failed to retrieve usage data. Ensure the project is initialized.")
	}
	return usage, nil
}

func (s *Server) handleApprovePlan(ctx context.Context, args struct{}) (string, error) {
	err := s.planSvc.ApprovePlan()
	if err != nil {
		return "", mcpErr("Failed to approve plan. Ensure a plan has been generated.")
	}
	return "Plan approved successfully", nil
}

func (s *Server) handleExplainSpec(ctx context.Context, args struct{}) (string, error) {
	result, err := s.aiSvc.ExplainSpec(ctx)
	if err != nil {
		return "", mcpErr("Failed to explain spec. Check your AI provider configuration.")
	}
	return result, nil
}

type QueryArgs struct {
	Question string `json:"question" jsonschema:"description=A natural language question about the project"`
}

func (s *Server) handleQuery(ctx context.Context, args QueryArgs) (string, error) {
	if args.Question == "" {
		return "", mcpErr("A question is required.")
	}
	answer, err := s.aiSvc.QueryProject(ctx, args.Question)
	if err != nil {
		return "", mcpErr("Failed to answer query. Check your AI provider configuration.")
	}
	return answer, nil
}

func (s *Server) handleSuggestPriorities(ctx context.Context, args struct{}) (any, error) {
	suggestions, err := s.aiSvc.SuggestPriorities(ctx)
	if err != nil {
		return nil, mcpErr("Failed to suggest priorities. Check your AI provider configuration and ensure a plan exists.")
	}
	return suggestions, nil
}

func (s *Server) handleReviewSpec(ctx context.Context, args struct{}) (any, error) {
	review, err := s.aiSvc.ReviewSpec(ctx)
	if err != nil {
		return nil, mcpErr("Failed to review spec. Check your AI provider configuration.")
	}
	return review, nil
}

type TransitionTaskArgs struct {
	TaskID   string `json:"task_id" jsonschema:"description=The ID of the task to transition"`
	Event    string `json:"event" jsonschema:"description=The transition event (start, complete, block, stop, unblock, reopen)"`
	Evidence string `json:"evidence,omitempty" jsonschema:"description=Optional evidence for the transition (e.g. commit hash)"`
	Actor    string `json:"actor,omitempty" jsonschema:"description=The actor performing the transition (defaults to ai-agent)"`
}

type AssignTaskArgs struct {
	TaskID   string `json:"task_id" jsonschema:"description=The ID of the task to assign"`
	Assignee string `json:"assignee" jsonschema:"description=The person or agent to assign the task to"`
}

// FlexBool accepts both boolean and string ("true"/"false") JSON values.
// MCP clients sometimes send string values for boolean fields.
type FlexBool bool

func (fb *FlexBool) UnmarshalJSON(data []byte) error {
	// Try bool first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fb = FlexBool(b)
		return nil
	}
	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fb = FlexBool(s == "true" || s == "1" || s == "yes")
		return nil
	}
	return fmt.Errorf("expected boolean or string, got %s", string(data))
}

// FlexInt accepts both integer and string JSON values.
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*fi = FlexInt(i)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		var n int
		if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
			*fi = FlexInt(n)
			return nil
		}
	}
	return fmt.Errorf("expected integer or string, got %s", string(data))
}

// StatusArgs defines filter parameters for roady_status tool
type StatusArgs struct {
	Status   string   `json:"status,omitempty" jsonschema:"description=Filter by status (comma-separated: pending,blocked,in_progress,done,verified)"`
	Priority string   `json:"priority,omitempty" jsonschema:"description=Filter by priority (comma-separated: high,medium,low)"`
	Ready    FlexBool `json:"ready,omitempty" jsonschema:"description=Show only tasks ready to start (unlocked + pending)"`
	Blocked  FlexBool `json:"blocked,omitempty" jsonschema:"description=Show only blocked tasks"`
	Active   FlexBool `json:"active,omitempty" jsonschema:"description=Show only in-progress tasks"`
	Limit    FlexInt  `json:"limit,omitempty" jsonschema:"description=Limit number of tasks returned"`
	JSON     FlexBool `json:"json,omitempty" jsonschema:"description=Return structured JSON output instead of text"`
}

func (s *Server) handleAssignTask(ctx context.Context, args AssignTaskArgs) (string, error) {
	err := s.taskSvc.AssignTask(ctx, args.TaskID, args.Assignee)
	if err != nil {
		return "", mcpErr(fmt.Sprintf("Failed to assign task '%s' to '%s'. Ensure the task exists in the plan.", args.TaskID, args.Assignee))
	}
	return fmt.Sprintf("Task %s assigned to %s", args.TaskID, args.Assignee), nil
}

func (s *Server) handleTransitionTask(ctx context.Context, args TransitionTaskArgs) (string, error) {
	actor := args.Actor
	if actor == "" {
		actor = "ai-agent"
	}
	err := s.taskSvc.TransitionTask(args.TaskID, args.Event, actor, args.Evidence)
	if err != nil {
		return "", mcpErr(fmt.Sprintf("Failed to transition task '%s' with event '%s'. Ensure the task exists and the transition is valid.", args.TaskID, args.Event))
	}
	return fmt.Sprintf("Task %s transitioned with event %s successfully", args.TaskID, args.Event), nil
}

func (s *Server) handleInit(ctx context.Context, args InitArgs) (string, error) {
	err := s.initSvc.InitializeProject(args.Name)
	if err != nil {
		return "", mcpErr("Failed to initialize project. Check directory permissions and ensure the name is valid.")
	}
	return fmt.Sprintf("Project %s initialized successfully", args.Name), nil
}

func (s *Server) handleGetSpec(ctx context.Context, args struct{}) (any, error) {
	spec, err := s.specSvc.GetSpec()
	if err != nil {
		return nil, mcpErr("Failed to load spec. Ensure the project is initialized with 'roady init'.")
	}
	return spec, nil
}

func (s *Server) handleGetPlan(ctx context.Context, args struct{}) (any, error) {
	plan, err := s.planSvc.GetPlan()
	if err != nil {
		return nil, mcpErr("Failed to load plan. Generate a plan first with 'roady plan generate'.")
	}
	return plan, nil
}

func (s *Server) handleGetState(ctx context.Context, args struct{}) (any, error) {
	state, err := s.planSvc.GetState()
	if err != nil {
		return nil, mcpErr("Failed to load execution state. Ensure a plan has been generated.")
	}
	return state, nil
}

func (s *Server) handleGeneratePlan(ctx context.Context, args struct{}) (string, error) {
	plan, err := s.planSvc.GeneratePlan(ctx)
	if err != nil {
		return "", mcpErr("Failed to generate plan. Ensure a spec exists with at least one feature.")
	}
	return fmt.Sprintf("Plan generated with %d tasks. Plan ID: %s", len(plan.Tasks), plan.ID), nil
}

func (s *Server) handleUpdatePlan(ctx context.Context, args UpdatePlanArgs) (string, error) {
	plan, err := s.planSvc.UpdatePlan(args.Tasks)
	if err != nil {
		return "", mcpErr("Failed to update plan. Ensure the task list is valid and a spec exists.")
	}
	return fmt.Sprintf("Plan updated with %d tasks. Plan ID: %s", len(plan.Tasks), plan.ID), nil
}

func (s *Server) handleDetectDrift(ctx context.Context, args struct{}) (any, error) {
	report, err := s.driftSvc.DetectDrift(ctx)
	if err != nil {
		return nil, mcpErr("Failed to detect drift. Ensure both spec and plan exist.")
	}
	return report, nil
}

func (s *Server) handleStatus(ctx context.Context, args StatusArgs) (any, error) {
	plan, err := s.planSvc.GetPlan()
	if err != nil {
		return nil, mcpErr("Failed to load plan. Generate a plan first with 'roady plan generate'.")
	}
	if plan == nil {
		return "No plan found. Run roady_generate_plan first.", nil
	}

	state, err := s.planSvc.GetState()
	if err != nil {
		return nil, mcpErr("Failed to load execution state. Ensure a plan has been generated.")
	}
	if state == nil {
		return "No execution state found.", nil
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
		if bool(args.Ready) {
			if status != planning.StatusPending || !isTaskUnlockedByDeps(t, state) {
				continue
			}
		}
		if bool(args.Blocked) && status != planning.StatusBlocked {
			continue
		}
		if bool(args.Active) && status != planning.StatusInProgress {
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
	if int(args.Limit) > 0 && len(filtered) > int(args.Limit) {
		filtered = filtered[:int(args.Limit)]
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
	if bool(args.JSON) {
		type taskOutput struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Status   string `json:"status"`
			Priority string `json:"priority"`
			Owner    string `json:"owner,omitempty"`
			Unlocked bool   `json:"unlocked,omitempty"`
		}

		tasks := make([]taskOutput, 0, len(filtered))
		for _, t := range filtered {
			status := planning.StatusPending
			owner := ""
			if res, ok := state.TaskStates[t.ID]; ok {
				status = res.Status
				owner = res.Owner
			}
			tasks = append(tasks, taskOutput{
				ID:       t.ID,
				Title:    t.Title,
				Status:   string(status),
				Priority: string(t.Priority),
				Owner:    owner,
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
			return nil, mcpErr("Failed to format status output.")
		}
		return string(jsonBytes), nil
	}

	// Text output
	statusStr := fmt.Sprintf("Tasks: %d total\n- Done: %d\n- In Progress: %d\n- Pending: %d\n- Blocked: %d",
		len(plan.Tasks), counts["done"]+counts["verified"], counts["in_progress"], counts["pending"], counts["blocked"])

	if len(statusFilters) > 0 || len(priorityFilters) > 0 || bool(args.Ready) || bool(args.Blocked) || bool(args.Active) {
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
		return nil, mcpErr("Failed to check policy compliance. Ensure a policy.yaml and plan exist.")
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
		return nil, mcpErr("Failed to list dependencies. Ensure .roady/deps.yaml exists.")
	}
	return deps, nil
}

func (s *Server) handleDepsScan(ctx context.Context, args struct{}) (any, error) {
	result, err := s.depSvc.ScanDependentRepos(nil)
	if err != nil {
		return nil, mcpErr("Failed to scan dependent repositories. Check that dependency paths are valid.")
	}
	return result, nil
}

type DepsGraphArgs struct {
	CheckCycles bool `json:"check_cycles,omitempty" jsonschema:"description=Whether to check for cyclic dependencies"`
}

func (s *Server) handleDepsGraph(ctx context.Context, args DepsGraphArgs) (any, error) {
	summary, err := s.depSvc.GetDependencySummary()
	if err != nil {
		return nil, mcpErr("Failed to get dependency summary. Ensure .roady/deps.yaml exists.")
	}

	response := map[string]any{
		"summary": summary,
	}

	if args.CheckCycles {
		hasCycle, err := s.depSvc.CheckForCycles()
		if err != nil {
			return nil, mcpErr("Failed to check for dependency cycles.")
		}
		response["has_cycle"] = hasCycle
	}

	return response, nil
}

// Debt MCP handlers

func (s *Server) handleDebtReport(ctx context.Context, args struct{}) (any, error) {
	report, err := s.debtSvc.GetDebtReport(ctx)
	if err != nil {
		return nil, mcpErr("Failed to generate debt report. Ensure drift detection has been run.")
	}
	return report, nil
}

func (s *Server) handleDebtSummary(ctx context.Context, args struct{}) (any, error) {
	summary, err := s.debtSvc.GetDebtSummary(ctx)
	if err != nil {
		return nil, mcpErr("Failed to get debt summary. Ensure drift detection has been run.")
	}
	return summary, nil
}

func (s *Server) handleStickyDrift(ctx context.Context, args struct{}) (any, error) {
	items, err := s.debtSvc.GetStickyDrift()
	if err != nil {
		return nil, mcpErr("Failed to get sticky drift items. Ensure drift history exists.")
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
		return nil, mcpErr("Failed to get debt trend. Ensure drift history exists.")
	}
	return trend, nil
}

// Coordinator-based snapshot and task query handlers (v0.6.0)

func (s *Server) handleGetSnapshot(ctx context.Context, args struct{}) (any, error) {
	snapshot, err := s.planSvc.GetProjectSnapshot(ctx)
	if err != nil {
		return nil, mcpErr("Failed to get project snapshot. Ensure a plan and state exist.")
	}

	type snapshotResp struct {
		Progress      float64  `json:"progress"`
		UnlockedTasks []string `json:"unlocked_tasks"`
		BlockedTasks  []string `json:"blocked_tasks"`
		InProgress    []string `json:"in_progress"`
		Completed     []string `json:"completed"`
		Verified      []string `json:"verified"`
		TotalTasks    int      `json:"total_tasks"`
		SnapshotTime  string   `json:"snapshot_time"`
	}

	totalTasks := 0
	if snapshot.Plan != nil {
		totalTasks = len(snapshot.Plan.Tasks)
	}

	return snapshotResp{
		Progress:      snapshot.Progress,
		UnlockedTasks: orEmpty(snapshot.UnlockedTasks),
		BlockedTasks:  orEmpty(snapshot.BlockedTasks),
		InProgress:    orEmpty(snapshot.InProgress),
		Completed:     orEmpty(snapshot.Completed),
		Verified:      orEmpty(snapshot.Verified),
		TotalTasks:    totalTasks,
		SnapshotTime:  snapshot.SnapshotTime.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *Server) handleGetReadyTasks(ctx context.Context, args struct{}) (any, error) {
	tasks, err := s.planSvc.GetReadyTasks(ctx)
	if err != nil {
		return nil, mcpErr("Failed to get ready tasks. Ensure a plan and state exist.")
	}
	return tasks, nil
}

func (s *Server) handleGetBlockedTasks(ctx context.Context, args struct{}) (any, error) {
	tasks, err := s.planSvc.GetBlockedTasks(ctx)
	if err != nil {
		return nil, mcpErr("Failed to get blocked tasks. Ensure a plan and state exist.")
	}
	return tasks, nil
}

func (s *Server) handleGetInProgressTasks(ctx context.Context, args struct{}) (any, error) {
	tasks, err := s.planSvc.GetInProgressTasks(ctx)
	if err != nil {
		return nil, mcpErr("Failed to get in-progress tasks. Ensure a plan and state exist.")
	}
	return tasks, nil
}

// --- Workspace Sync Handlers ---

func (s *Server) handleWorkspacePush(ctx context.Context, args struct{}) (any, error) {
	svc := application.NewWorkspaceSyncService(s.root, s.auditSvc)
	result, err := svc.Push(ctx)
	if err != nil {
		return nil, mcpErr(fmt.Sprintf("workspace push failed: %s", err))
	}
	return result, nil
}

func (s *Server) handleWorkspacePull(ctx context.Context, args struct{}) (any, error) {
	svc := application.NewWorkspaceSyncService(s.root, s.auditSvc)
	result, err := svc.Pull(ctx)
	if err != nil {
		return nil, mcpErr(fmt.Sprintf("workspace pull failed: %s", err))
	}
	return result, nil
}

// --- Smart Decompose Handler ---

func (s *Server) handleSmartDecompose(ctx context.Context, args struct{}) (any, error) {
	if s.aiSvc == nil {
		return nil, mcpErr("AI service not available")
	}
	result, err := s.aiSvc.SmartDecompose(ctx, s.root)
	if err != nil {
		return nil, mcpErr(fmt.Sprintf("smart decompose failed: %s", err))
	}
	return result, nil
}

// --- Team Handlers ---

type TeamAddArgs struct {
	Name string `json:"name" jsonschema:"description=The name of the team member"`
	Role string `json:"role" jsonschema:"description=The role: admin, member, or viewer"`
}

type TeamRemoveArgs struct {
	Name string `json:"name" jsonschema:"description=The name of the team member to remove"`
}

func (s *Server) handleTeamList(ctx context.Context, args struct{}) (any, error) {
	cfg, err := s.teamSvc.ListMembers()
	if err != nil {
		return nil, mcpErr("failed to list team members")
	}
	return cfg, nil
}

func (s *Server) handleTeamAdd(ctx context.Context, args TeamAddArgs) (string, error) {
	if args.Name == "" {
		return "", mcpErr("name is required")
	}
	if args.Role == "" {
		return "", mcpErr("role is required")
	}
	if err := s.teamSvc.AddMember(args.Name, teamRole(args.Role)); err != nil {
		return "", mcpErr(fmt.Sprintf("failed to add member: %s", err))
	}
	return fmt.Sprintf("Member %s added with role %s", args.Name, args.Role), nil
}

func (s *Server) handleTeamRemove(ctx context.Context, args TeamRemoveArgs) (string, error) {
	if args.Name == "" {
		return "", mcpErr("name is required")
	}
	if err := s.teamSvc.RemoveMember(args.Name); err != nil {
		return "", mcpErr(fmt.Sprintf("failed to remove member: %s", err))
	}
	return fmt.Sprintf("Member %s removed", args.Name), nil
}

func teamRole(s string) team.Role {
	return team.Role(s)
}

// orEmpty returns the slice or an empty slice if nil (for clean JSON output).
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
