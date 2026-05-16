// Package dashboard provides a web-based UI for Roady project management.
package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

//go:embed templates/*
var templatesFS embed.FS

// DataProvider provides data for the dashboard.
type DataProvider interface {
	GetPlan() (*planning.Plan, error)
	GetState() (*planning.ExecutionState, error)
}

// Server is the dashboard HTTP server.
type Server struct {
	addr     string
	provider DataProvider
	server   *http.Server
	tmpl     *template.Template

	// Optional cross-project Kanban wiring. When set, /org/kanban routes are
	// registered. See EnableOrgKanban.
	orgProvider   OrgKanbanProvider
	orgRepoOpener repoOpener

	// Optional task-action wiring. When set, POST /actions/task/* routes are
	// registered and the Kanban board renders action buttons. See EnableTaskActions.
	taskActions    TaskActions
	orgTaskActions OrgTaskActions

	// Optional SSE hub for live updates. Created on first /events subscription.
	sse *sseHub

	// Optional bearer-token gate for every request. When empty, the server is
	// public. See EnableAuthToken.
	authToken string
}

// NewServer creates a new dashboard server.
func NewServer(addr string, provider DataProvider) (*Server, error) {
	funcMap := template.FuncMap{
		"statusClass": statusClass,
		"formatTime":  formatTime,
		"json":        toJSON,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Server{
		addr:     addr,
		provider: provider,
		tmpl:     tmpl,
	}, nil
}

// Start starts the dashboard server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /plan", s.handlePlan)
	mux.HandleFunc("GET /tasks", s.handleTasks)
	mux.HandleFunc("GET /kanban", s.handleKanban)
	mux.HandleFunc("GET /api/plan", s.handleAPIPlan)
	mux.HandleFunc("GET /api/state", s.handleAPIState)
	mux.HandleFunc("GET /api/kanban", s.handleAPIKanban)

	if s.orgProvider != nil {
		mux.HandleFunc("GET /org/kanban", s.orgKanbanHandler(s.orgProvider, s.orgRepoOpener))
		mux.HandleFunc("GET /api/org/kanban", s.orgKanbanAPIHandler(s.orgProvider, s.orgRepoOpener))
	}

	if s.taskActions != nil {
		mux.HandleFunc("POST /actions/task/start", s.handleTaskStart)
		mux.HandleFunc("POST /actions/task/complete", s.handleTaskComplete)
		mux.HandleFunc("POST /actions/task/block", s.handleTaskBlock)
		mux.HandleFunc("POST /actions/task/unblock", s.handleTaskUnblock)
		mux.HandleFunc("POST /actions/task/reopen", s.handleTaskReopen)
	}

	// SSE live-update stream. Always registered; clients reconnect on disconnect.
	if s.sse == nil {
		s.sse = newSSEHub()
	}
	mux.HandleFunc("GET /events", s.handleEvents)

	handler := http.Handler(mux)
	if s.authToken != "" {
		handler = authMiddleware(s.authToken, handler)
	}

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // SSE needs no write timeout
	}

	log.Printf("Dashboard server starting on %s", s.addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// PageData holds data for template rendering.
type PageData struct {
	Title string
	Plan  *planning.Plan
	State *planning.ExecutionState
	Tasks []TaskView
	Stats DashboardStats
	Error string
}

// TaskView combines task and state info for display.
type TaskView struct {
	Task         planning.Task
	Status       planning.TaskStatus
	Owner        string
	HasLinks     bool
	ProjectLabel string // set on cross-project Kanban cards; empty for per-project views
	ProjectPath  string // workspace root, set on org-kanban cards so actions can route to it
	ProjectName  string // sub-project name (empty = root project), set on org-kanban cards
}

// DashboardStats holds summary statistics.
type DashboardStats struct {
	TotalTasks int
	Pending    int
	InProgress int
	Done       int
	Blocked    int
	Completion float64
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Dashboard"}

	plan, err := s.provider.GetPlan()
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Plan = plan
	}

	state, _ := s.provider.GetState()
	data.State = state

	if plan != nil && state != nil {
		data.Stats = calculateStats(plan, state)
		data.Tasks = buildTaskViews(plan, state)
	}

	s.render(w, "index.html", data)
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Plan"}

	plan, err := s.provider.GetPlan()
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Plan = plan
	}

	s.render(w, "plan.html", data)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Tasks"}

	plan, err := s.provider.GetPlan()
	if err != nil {
		data.Error = err.Error()
		s.render(w, "tasks.html", data)
		return
	}

	state, _ := s.provider.GetState()
	if plan != nil {
		data.Tasks = buildTaskViews(plan, state)
		if state != nil {
			data.Stats = calculateStats(plan, state)
		}
	}

	s.render(w, "tasks.html", data)
}

func (s *Server) handleAPIPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := s.provider.GetPlan()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(plan)
}

func (s *Server) handleAPIState(w http.ResponseWriter, r *http.Request) {
	state, err := s.provider.GetState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func calculateStats(plan *planning.Plan, state *planning.ExecutionState) DashboardStats {
	stats := DashboardStats{
		TotalTasks: len(plan.Tasks),
	}

	for _, task := range plan.Tasks {
		status := planning.StatusPending
		if result, ok := state.TaskStates[task.ID]; ok {
			status = result.Status
		}

		switch status {
		case planning.StatusPending:
			stats.Pending++
		case planning.StatusInProgress:
			stats.InProgress++
		case planning.StatusDone, planning.StatusVerified:
			stats.Done++
		case planning.StatusBlocked:
			stats.Blocked++
		}
	}

	if stats.TotalTasks > 0 {
		stats.Completion = float64(stats.Done) / float64(stats.TotalTasks) * 100
	}

	return stats
}

func buildTaskViews(plan *planning.Plan, state *planning.ExecutionState) []TaskView {
	views := make([]TaskView, 0, len(plan.Tasks))

	for _, task := range plan.Tasks {
		view := TaskView{Task: task, Status: planning.StatusPending}

		if state != nil {
			if result, ok := state.TaskStates[task.ID]; ok {
				view.Status = result.Status
				view.Owner = result.Owner
				view.HasLinks = len(result.ExternalRefs) > 0
			}
		}

		views = append(views, view)
	}

	return views
}

// Template helper functions
func statusClass(status planning.TaskStatus) string {
	switch status {
	case planning.StatusPending:
		return "status-pending"
	case planning.StatusInProgress:
		return "status-progress"
	case planning.StatusDone, planning.StatusVerified:
		return "status-done"
	case planning.StatusBlocked:
		return "status-blocked"
	default:
		return "status-unknown"
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
