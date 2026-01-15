package mcp

import (
	"context"
	"fmt"

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

func (s *Server) handleStatus(ctx context.Context, args struct{}) (string, error) {
	plan, err := s.planSvc.GetPlan()
	if err != nil {
		return "", fmt.Errorf("failed to load plan: %w", err)
	}

	state, err := s.planSvc.GetState()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	counts := make(map[string]int)
	for _, t := range plan.Tasks {
		status := "pending"
		if res, ok := state.TaskStates[t.ID]; ok {
			status = string(res.Status)
		}
		counts[status]++
	}

	statusStr := fmt.Sprintf("Tasks: %d total\n- Done: %d\n- In Progress: %d\n- Pending: %d\n- Blocked: %d",
		len(plan.Tasks), counts["done"], counts["in_progress"], counts["pending"], counts["blocked"])
	return statusStr, nil
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
