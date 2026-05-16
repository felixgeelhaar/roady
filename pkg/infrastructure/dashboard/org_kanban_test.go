package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// stubOrgProvider implements OrgKanbanProvider for tests.
type stubOrgProvider struct {
	items []application.DiscoveredProject
}

func (s *stubOrgProvider) DiscoverProjectsWithSub() ([]application.DiscoveredProject, error) {
	return s.items, nil
}

// initSubProject creates a sub-project with a plan + state. Returns the
// discovered-project descriptor matching it.
func initSubProject(t *testing.T, root, name string, tasks []planning.Task, states map[string]planning.TaskResult) application.DiscoveredProject {
	t.Helper()
	repo, err := storage.NewFilesystemRepositoryForProject(root, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveSpec(&spec.ProductSpec{ID: "s-" + name, Title: name}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SavePlan(&planning.Plan{ID: "p-" + name, Tasks: tasks}); err != nil {
		t.Fatal(err)
	}
	if states != nil {
		if err := repo.SaveState(&planning.ExecutionState{TaskStates: states}); err != nil {
			t.Fatal(err)
		}
	}
	return application.DiscoveredProject{Path: root, SubProject: name}
}

func TestBuildOrgKanbanBoard_MergesAcrossProjects(t *testing.T) {
	root := t.TempDir()

	auth := initSubProject(t, root, "auth",
		[]planning.Task{
			{ID: "auth-1", Title: "ready"},
			{ID: "auth-2", Title: "wip"},
		},
		map[string]planning.TaskResult{
			"auth-2": {Status: planning.StatusInProgress, Owner: "alice"},
		},
	)
	billing := initSubProject(t, root, "billing",
		[]planning.Task{
			{ID: "bill-1", Title: "done"},
			{ID: "bill-2", Title: "blocked"},
		},
		map[string]planning.TaskResult{
			"bill-1": {Status: planning.StatusDone},
			"bill-2": {Status: planning.StatusBlocked},
		},
	)

	prov := &stubOrgProvider{items: []application.DiscoveredProject{auth, billing}}
	board := buildOrgKanbanBoard(prov, defaultRepoOpener)

	if board.TotalTasks != 4 {
		t.Errorf("TotalTasks = %d, want 4", board.TotalTasks)
	}
	if board.TotalDone != 1 {
		t.Errorf("TotalDone = %d, want 1", board.TotalDone)
	}
	if len(board.Projects) != 2 {
		t.Fatalf("Projects = %d, want 2", len(board.Projects))
	}

	wantCount := map[string]int{
		"backlog":                         0,
		"ready":                           1, // auth-1
		string(planning.StatusInProgress): 1,
		string(planning.StatusBlocked):    1,
		string(planning.StatusDone):       1,
	}
	for _, c := range board.Columns {
		if got, want := c.Count, wantCount[c.Status]; got != want {
			t.Errorf("column %q count = %d, want %d", c.Status, got, want)
		}
	}

	// Each card carries a project label.
	for _, c := range board.Columns {
		for _, tv := range c.Tasks {
			if tv.ProjectLabel == "" {
				t.Errorf("card %s missing ProjectLabel", tv.Task.ID)
			}
		}
	}
}

func TestBuildOrgKanbanBoard_ProjectLabelIncludesSubProject(t *testing.T) {
	root := t.TempDir()
	d := initSubProject(t, root, "auth", []planning.Task{{ID: "x"}}, nil)

	prov := &stubOrgProvider{items: []application.DiscoveredProject{d}}
	board := buildOrgKanbanBoard(prov, defaultRepoOpener)

	if len(board.Projects) != 1 {
		t.Fatalf("Projects = %d, want 1", len(board.Projects))
	}
	wantLabel := filepath.Base(root) + "/auth"
	if board.Projects[0].Label != wantLabel {
		t.Errorf("Project label = %q, want %q", board.Projects[0].Label, wantLabel)
	}
}

func TestBuildOrgKanbanBoard_MissingPlanCountsProjectButNoTasks(t *testing.T) {
	root := t.TempDir()
	// Project initialised but no plan saved.
	repo, err := storage.NewFilesystemRepositoryForProject(root, "empty")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Initialize(); err != nil {
		t.Fatal(err)
	}
	d := application.DiscoveredProject{Path: root, SubProject: "empty"}

	prov := &stubOrgProvider{items: []application.DiscoveredProject{d}}
	board := buildOrgKanbanBoard(prov, defaultRepoOpener)

	if board.TotalTasks != 0 {
		t.Errorf("TotalTasks = %d, want 0", board.TotalTasks)
	}
	if len(board.Projects) != 1 {
		t.Errorf("Projects = %d, want 1", len(board.Projects))
	}
}

func TestBuildOrgKanbanBoard_DiscoveryError(t *testing.T) {
	prov := failingOrgProvider{}
	board := buildOrgKanbanBoard(prov, defaultRepoOpener)
	if board.Err == "" {
		t.Error("expected Err to be set on discovery failure")
	}
	if len(board.Columns) != 5 {
		t.Errorf("columns = %d, want 5 empty", len(board.Columns))
	}
}

type failingOrgProvider struct{}

func (failingOrgProvider) DiscoverProjectsWithSub() ([]application.DiscoveredProject, error) {
	return nil, os.ErrPermission
}

func TestOrgKanbanRoutesRegisteredWhenEnabled(t *testing.T) {
	root := t.TempDir()
	d := initSubProject(t, root, "auth",
		[]planning.Task{{ID: "auth-1", Title: "ready"}},
		nil,
	)

	srv, err := NewServer(":0", &kanbanStubProvider{plan: &planning.Plan{}, state: &planning.ExecutionState{}})
	if err != nil {
		t.Fatal(err)
	}
	srv.EnableOrgKanban(&stubOrgProvider{items: []application.DiscoveredProject{d}}, nil)

	// HTML
	rec := httptest.NewRecorder()
	srv.orgKanbanHandler(srv.orgProvider, srv.orgRepoOpener)(rec, httptest.NewRequest(http.MethodGet, "/org/kanban", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("html status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Org Kanban", "auth", "auth-1"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}

	// JSON
	rec = httptest.NewRecorder()
	srv.orgKanbanAPIHandler(srv.orgProvider, srv.orgRepoOpener)(rec, httptest.NewRequest(http.MethodGet, "/api/org/kanban", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("api status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("content-type = %q, want application/json", got)
	}
	var board OrgKanbanBoard
	if err := json.Unmarshal(rec.Body.Bytes(), &board); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(board.Projects) != 1 {
		t.Errorf("Projects = %d, want 1", len(board.Projects))
	}
}
