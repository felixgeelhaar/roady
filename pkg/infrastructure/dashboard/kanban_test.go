package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type kanbanStubProvider struct {
	plan  *planning.Plan
	state *planning.ExecutionState
	err   error
}

func (s *kanbanStubProvider) GetPlan() (*planning.Plan, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.plan, nil
}

func (s *kanbanStubProvider) GetState() (*planning.ExecutionState, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.state, nil
}

func sampleKanbanPlan() *planning.Plan {
	return &planning.Plan{
		ID: "plan-1",
		Tasks: []planning.Task{
			{ID: "a-ready", Title: "ready"},
			{ID: "b-progress", Title: "in progress"},
			{ID: "c-blocked", Title: "blocked"},
			{ID: "d-done", Title: "done"},
			{ID: "e-backlog", Title: "backlog (unmet dep)", DependsOn: []string{"f-missing"}},
			{ID: "g-ready-dep", Title: "ready (dep done)", DependsOn: []string{"d-done"}},
		},
	}
}

func sampleKanbanState() *planning.ExecutionState {
	return &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"b-progress": {Status: planning.StatusInProgress, Owner: "alice"},
			"c-blocked":  {Status: planning.StatusBlocked},
			"d-done":     {Status: planning.StatusDone},
		},
	}
}

func TestBuildKanbanBoard_ColumnOrderAndCounts(t *testing.T) {
	board := buildKanbanBoard(sampleKanbanPlan(), sampleKanbanState())

	if got, want := len(board.Columns), 5; got != want {
		t.Fatalf("columns = %d, want %d", got, want)
	}
	wantOrder := []string{"backlog", "ready", string(planning.StatusInProgress), string(planning.StatusBlocked), string(planning.StatusDone)}
	for i, want := range wantOrder {
		if board.Columns[i].Status != want {
			t.Errorf("column[%d].Status = %q, want %q", i, board.Columns[i].Status, want)
		}
	}

	wantCount := map[string]int{
		"backlog":                         1,
		"ready":                           2, // a-ready + g-ready-dep (dep d-done is done)
		string(planning.StatusInProgress): 1,
		string(planning.StatusBlocked):    1,
		string(planning.StatusDone):       1,
	}
	for _, c := range board.Columns {
		if got, want := c.Count, wantCount[c.Status]; got != want {
			t.Errorf("column %q count = %d, want %d", c.Status, got, want)
		}
	}
	if board.TotalTasks != len(sampleKanbanPlan().Tasks) {
		t.Errorf("TotalTasks = %d, want %d", board.TotalTasks, len(sampleKanbanPlan().Tasks))
	}
}

func TestBuildKanbanBoard_VerifiedRollsIntoDone(t *testing.T) {
	plan := &planning.Plan{Tasks: []planning.Task{{ID: "x", Title: "verified"}}}
	state := &planning.ExecutionState{TaskStates: map[string]planning.TaskResult{
		"x": {Status: planning.StatusVerified},
	}}
	board := buildKanbanBoard(plan, state)
	for _, c := range board.Columns {
		if c.Status == string(planning.StatusDone) {
			if c.Count != 1 {
				t.Errorf("done column count = %d, want 1", c.Count)
			}
			return
		}
	}
	t.Fatal("done column not found")
}

func TestBuildKanbanBoard_EmptyPlan(t *testing.T) {
	board := buildKanbanBoard(nil, nil)
	if len(board.Columns) != 5 {
		t.Fatalf("expected 5 empty columns, got %d", len(board.Columns))
	}
	for _, c := range board.Columns {
		if c.Count != 0 {
			t.Errorf("empty plan column %q count = %d, want 0", c.Status, c.Count)
		}
	}
}

func TestBuildKanbanBoard_StableTaskOrder(t *testing.T) {
	plan := &planning.Plan{Tasks: []planning.Task{
		{ID: "z", Title: "z"},
		{ID: "a", Title: "a"},
		{ID: "m", Title: "m"},
	}}
	board := buildKanbanBoard(plan, nil)
	var ready *KanbanColumn
	for i := range board.Columns {
		if board.Columns[i].Status == "ready" {
			ready = &board.Columns[i]
			break
		}
	}
	if ready == nil || ready.Count != 3 {
		t.Fatalf("ready column missing or wrong count: %+v", ready)
	}
	wantOrder := []string{"a", "m", "z"}
	for i, tv := range ready.Tasks {
		if tv.Task.ID != wantOrder[i] {
			t.Errorf("ready[%d] = %q, want %q", i, tv.Task.ID, wantOrder[i])
		}
	}
}

func TestKanbanHTMLHandler_RendersAllColumns(t *testing.T) {
	srv, err := NewServer(":0", &kanbanStubProvider{plan: sampleKanbanPlan(), state: sampleKanbanState()})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/kanban", nil)
	rec := httptest.NewRecorder()
	srv.handleKanban(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Backlog", "Ready", "In Progress", "Blocked", "Done", "auto-refresh 30s"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	if !strings.Contains(body, "col-in_progress") {
		t.Error("expected col-in_progress class in body")
	}
}

func TestKanbanHTMLHandler_DragDropMarkup(t *testing.T) {
	// Drag-and-drop attributes + JS only render when task actions are wired.
	withActions, err := NewServer(":0", &kanbanStubProvider{plan: sampleKanbanPlan(), state: sampleKanbanState()})
	if err != nil {
		t.Fatal(err)
	}
	withActions.EnableTaskActions(&fakeTaskActions{})

	rec := httptest.NewRecorder()
	withActions.handleKanban(rec, httptest.NewRequest(http.MethodGet, "/kanban", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`draggable="true"`,
		`data-task-id=`,
		`data-source=`,
		`data-status=`,
		`drag cards between columns`,
		`TRANSITIONS`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("with actions: body missing %q", want)
		}
	}

	// Without actions: read-only board, no drag scaffolding.
	readOnly, err := NewServer(":0", &kanbanStubProvider{plan: sampleKanbanPlan(), state: sampleKanbanState()})
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	readOnly.handleKanban(rec, httptest.NewRequest(http.MethodGet, "/kanban", nil))
	body = rec.Body.String()

	// `draggable="true"` also appears in the CSS selector — only check for the
	// markup-only artifacts (data attributes + JS scaffolding + UI hint).
	for _, banned := range []string{`data-task-id=`, `TRANSITIONS`, `drag cards between columns`} {
		if strings.Contains(body, banned) {
			t.Errorf("read-only board should NOT contain %q", banned)
		}
	}
}

func TestKanbanAPIHandler_ReturnsJSON(t *testing.T) {
	srv, err := NewServer(":0", &kanbanStubProvider{plan: sampleKanbanPlan(), state: sampleKanbanState()})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/kanban", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIKanban(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("content-type = %q, want application/json", got)
	}
	var board KanbanBoard
	if err := json.Unmarshal(rec.Body.Bytes(), &board); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(board.Columns) != 5 {
		t.Errorf("columns = %d, want 5", len(board.Columns))
	}
	if board.UpdatedAt.IsZero() || time.Since(board.UpdatedAt) > time.Second {
		t.Errorf("UpdatedAt looks stale: %v", board.UpdatedAt)
	}
}
