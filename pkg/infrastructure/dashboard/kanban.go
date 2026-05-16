package dashboard

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// KanbanBoard groups the plan's tasks into status columns for a Kanban view.
type KanbanBoard struct {
	Columns     []KanbanColumn `json:"columns"`
	ProjectName string         `json:"project_name,omitempty"`
	TotalTasks  int            `json:"total_tasks"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// KanbanColumn is one status lane on the board.
type KanbanColumn struct {
	Name   string     `json:"name"`
	Status string     `json:"status"` // matches statusClass()
	Tasks  []TaskView `json:"tasks"`
	Count  int        `json:"count"`
}

// kanbanColumnOrder defines the column order rendered on the board.
// Backlog → tasks with no state or status=pending AND with unmet dependencies.
// Ready   → pending tasks whose dependencies are all done/verified.
// Other columns map directly to TaskStatus values.
var kanbanColumnOrder = []struct {
	Name   string
	Status string
}{
	{"Backlog", "backlog"},
	{"Ready", "ready"},
	{"In Progress", string(planning.StatusInProgress)},
	{"Blocked", string(planning.StatusBlocked)},
	{"Done", string(planning.StatusDone)},
}

// buildKanbanBoard returns a board grouped by column, ordered by kanbanColumnOrder.
// Within each column tasks are sorted by ID for stable rendering.
func buildKanbanBoard(plan *planning.Plan, state *planning.ExecutionState) KanbanBoard {
	board := KanbanBoard{
		UpdatedAt: time.Now(),
	}
	if plan == nil {
		for _, c := range kanbanColumnOrder {
			board.Columns = append(board.Columns, KanbanColumn{Name: c.Name, Status: c.Status})
		}
		return board
	}

	// Index task status to compute readiness.
	taskStatus := map[string]planning.TaskStatus{}
	for _, t := range plan.Tasks {
		taskStatus[t.ID] = planning.StatusPending
	}
	if state != nil {
		for id, r := range state.TaskStates {
			taskStatus[id] = r.Status
		}
	}

	cols := map[string]*KanbanColumn{}
	for _, c := range kanbanColumnOrder {
		cols[c.Status] = &KanbanColumn{Name: c.Name, Status: c.Status}
	}

	for _, task := range plan.Tasks {
		view := TaskView{Task: task, Status: planning.StatusPending}
		if state != nil {
			if r, ok := state.TaskStates[task.ID]; ok {
				view.Status = r.Status
				view.Owner = r.Owner
				view.HasLinks = len(r.ExternalRefs) > 0
			}
		}

		var target string
		switch view.Status {
		case planning.StatusInProgress:
			target = string(planning.StatusInProgress)
		case planning.StatusBlocked:
			target = string(planning.StatusBlocked)
		case planning.StatusDone, planning.StatusVerified:
			target = string(planning.StatusDone)
		default:
			if dependenciesMet(task, taskStatus) {
				target = "ready"
			} else {
				target = "backlog"
			}
		}

		col := cols[target]
		if col == nil {
			// Defensive: unknown status falls into backlog.
			col = cols["backlog"]
		}
		col.Tasks = append(col.Tasks, view)
	}

	for _, c := range kanbanColumnOrder {
		col := cols[c.Status]
		sort.SliceStable(col.Tasks, func(i, j int) bool {
			return col.Tasks[i].Task.ID < col.Tasks[j].Task.ID
		})
		col.Count = len(col.Tasks)
		board.Columns = append(board.Columns, *col)
		board.TotalTasks += col.Count
	}
	return board
}

// dependenciesMet returns true when every task in DependsOn has reached a
// terminal positive state (done or verified). Unknown dependency IDs are
// treated as not met so the task stays in backlog.
func dependenciesMet(task planning.Task, byID map[string]planning.TaskStatus) bool {
	for _, dep := range task.DependsOn {
		s, ok := byID[dep]
		if !ok {
			return false
		}
		if s != planning.StatusDone && s != planning.StatusVerified {
			return false
		}
	}
	return true
}

// kanbanData wraps the board plus standard PageData so templates can share
// the layout with other dashboard pages.
type kanbanData struct {
	Title          string
	Board          KanbanBoard
	Error          string
	ActionsEnabled bool
}

func (s *Server) handleKanban(w http.ResponseWriter, r *http.Request) {
	data := kanbanData{Title: "Kanban", ActionsEnabled: s.taskActions != nil}

	plan, err := s.provider.GetPlan()
	if err != nil {
		data.Error = err.Error()
		s.render(w, "kanban.html", data)
		return
	}
	state, _ := s.provider.GetState()
	data.Board = buildKanbanBoard(plan, state)

	s.render(w, "kanban.html", data)
}

func (s *Server) handleAPIKanban(w http.ResponseWriter, r *http.Request) {
	plan, err := s.provider.GetPlan()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state, _ := s.provider.GetState()
	board := buildKanbanBoard(plan, state)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(board)
}
